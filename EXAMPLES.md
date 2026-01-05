# Usage Examples

Complete examples demonstrating how to integrate and use the throttle library in real-world applications.

## Table of Contents

- [Basic HTTP API Rate Limiting](#basic-http-api-rate-limiting)
- [gRPC Service Rate Limiting](#grpc-service-rate-limiting)
- [Multi-Tier Rate Limiting](#multi-tier-rate-limiting)
- [Per-Endpoint Configuration](#per-endpoint-configuration)
- [Redis Backend (Distributed)](#redis-backend-distributed)
- [Monitoring and Observability](#monitoring-and-observability)

## Basic HTTP API Rate Limiting

### Simple Middleware

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/ASHISH26940/throttle/internal/algorithm"
    "github.com/ASHISH26940/throttle/internal/errors"
    "github.com/ASHISH26940/throttle/internal/types"
)

func main() {
    // Create rate limiter
    cfg := types.Config{
        Rate:   100,              // 100 requests
        Window: time.Minute,      // per minute
        Burst:  120,              // allow bursts up to 120
    }

    limiter, err := algorithm.NewTokenBucket(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer limiter.Close()

    // Wrap handlers with rate limiting
    http.Handle("/api/", rateLimitMiddleware(limiter)(apiHandler()))
    http.Handle("/public/", rateLimitMiddleware(limiter)(publicHandler()))

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func rateLimitMiddleware(limiter algorithm.Algorithm) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Use client IP as the key
            key := getClientIP(r)

            err := limiter.Allow(key)
            if err != nil {
                handleRateLimitError(w, err)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func handleRateLimitError(w http.ResponseWriter, err error) {
    if rlErr, ok := err.(*errors.RateLimitError); ok {
        // Set standard rate limit headers
        w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rlErr.Limit))
        w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rlErr.Remaining))
        w.Header().Set("Retry-After", fmt.Sprintf("%.0f", rlErr.RetryAfter.Seconds()))
        w.Header().Set("X-RateLimit-Reset", rlErr.ResetAt.Format(time.RFC3339))

        w.WriteHeader(http.StatusTooManyRequests)
        fmt.Fprintf(w, `{"error": "rate_limit_exceeded", "retry_after": %d}`,
            int(rlErr.RetryAfter.Seconds()))
        return
    }

    // Generic error
    http.Error(w, "Internal error", http.StatusInternalServerError)
}

func getClientIP(r *http.Request) string {
    // Check X-Forwarded-For header
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        return xff
    }
    // Check X-Real-IP header
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    return r.RemoteAddr
}

func apiHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "API response")
    })
}

func publicHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Public response")
    })
}
```

### Per-User Rate Limiting with Authentication

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/ASHISH26940/throttle/internal/algorithm"
    "github.com/ASHISH26940/throttle/internal/types"
)

type contextKey string

const userIDKey contextKey = "userID"

func main() {
    cfg := types.Config{
        Rate:   1000,
        Window: time.Hour,
        Burst:  1200,
    }

    limiter, _ := algorithm.NewTokenBucket(cfg)
    defer limiter.Close()

    // Chain middleware: auth -> rate limit -> handler
    handler := authMiddleware(
        userRateLimitMiddleware(limiter)(
            protectedHandler(),
        ),
    )

    http.Handle("/api/protected", handler)
    http.ListenAndServe(":8080", nil)
}

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Missing authorization", http.StatusUnauthorized)
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        userID, err := validateToken(token)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        // Add userID to context
        ctx := context.WithValue(r.Context(), userIDKey, userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func userRateLimitMiddleware(limiter algorithm.Algorithm) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get userID from context
            userID, ok := r.Context().Value(userIDKey).(string)
            if !ok {
                http.Error(w, "No user context", http.StatusInternalServerError)
                return
            }

            // Rate limit by user ID
            if err := limiter.Allow(userID); err != nil {
                handleRateLimitError(w, err)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func validateToken(token string) (string, error) {
    // Your JWT/OAuth validation logic here
    return "user123", nil
}

func protectedHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := r.Context().Value(userIDKey).(string)
        fmt.Fprintf(w, "Protected resource for user: %s\n", userID)
    })
}
```

## gRPC Service Rate Limiting

```go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"

    "github.com/ASHISH26940/throttle/internal/algorithm"
    "github.com/ASHISH26940/throttle/internal/errors"
    "github.com/ASHISH26940/throttle/internal/types"
)

// gRPC Unary Interceptor
func RateLimitUnaryInterceptor(limiter algorithm.Algorithm) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // Extract client identifier from context
        clientID, err := getClientIDFromContext(ctx)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "missing client ID")
        }

        // Check rate limit
        if err := limiter.Allow(clientID); err != nil {
            return nil, handleGRPCRateLimitError(err)
        }

        // Continue to handler
        return handler(ctx, req)
    }
}

// gRPC Stream Interceptor
func RateLimitStreamInterceptor(limiter algorithm.Algorithm) grpc.StreamServerInterceptor {
    return func(
        srv interface{},
        ss grpc.ServerStream,
        info *grpc.StreamServerInfo,
        handler grpc.StreamHandler,
    ) error {
        clientID, err := getClientIDFromContext(ss.Context())
        if err != nil {
            return status.Error(codes.Unauthenticated, "missing client ID")
        }

        if err := limiter.Allow(clientID); err != nil {
            return handleGRPCRateLimitError(err)
        }

        return handler(srv, ss)
    }
}

func getClientIDFromContext(ctx context.Context) (string, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return "", status.Error(codes.InvalidArgument, "missing metadata")
    }

    clientIDs := md.Get("client-id")
    if len(clientIDs) == 0 {
        return "", status.Error(codes.InvalidArgument, "missing client-id")
    }

    return clientIDs[0], nil
}

func handleGRPCRateLimitError(err error) error {
    if rlErr, ok := err.(*errors.RateLimitError); ok {
        // Create gRPC status with metadata
        st := status.New(codes.ResourceExhausted, "rate limit exceeded")

        // Add retry-after to status details
        st, _ = st.WithDetails(&status.Status{
            Code:    int32(codes.ResourceExhausted),
            Message: "rate limit exceeded",
        })

        return st.Err()
    }

    return status.Error(codes.Internal, "internal error")
}

func main() {
    cfg := types.Config{
        Rate:   100,
        Window: time.Minute,
        Burst:  150,
    }

    limiter, _ := algorithm.NewTokenBucket(cfg)
    defer limiter.Close()

    opts := []grpc.ServerOption{
        grpc.UnaryInterceptor(RateLimitUnaryInterceptor(limiter)),
        grpc.StreamInterceptor(RateLimitStreamInterceptor(limiter)),
    }

    server := grpc.NewServer(opts...)
    // Register your services...

    log.Println("gRPC server starting on :50051")
    // server.Serve(listener)
}
```

## Multi-Tier Rate Limiting

Different limits for different user tiers (free, premium, enterprise):

```go
package main

import (
    "fmt"
    "net/http"
    "time"

    "github.com/ASHISH26940/throttle/internal/algorithm"
    "github.com/ASHISH26940/throttle/internal/types"
)

type UserTier string

const (
    TierFree       UserTier = "free"
    TierPremium    UserTier = "premium"
    TierEnterprise UserTier = "enterprise"
)

type TieredRateLimiter struct {
    limiters map[UserTier]algorithm.Algorithm
}

func NewTieredRateLimiter() *TieredRateLimiter {
    return &TieredRateLimiter{
        limiters: map[UserTier]algorithm.Algorithm{
            TierFree: mustCreateLimiter(types.Config{
                Rate:   10,
                Window: time.Minute,
                Burst:  15,
            }),
            TierPremium: mustCreateLimiter(types.Config{
                Rate:   100,
                Window: time.Minute,
                Burst:  150,
            }),
            TierEnterprise: mustCreateLimiter(types.Config{
                Rate:   1000,
                Window: time.Minute,
                Burst:  1500,
            }),
        },
    }
}

func (trl *TieredRateLimiter) Allow(tier UserTier, userID string) error {
    limiter, ok := trl.limiters[tier]
    if !ok {
        limiter = trl.limiters[TierFree] // Default to free tier
    }

    return limiter.Allow(userID)
}

func (trl *TieredRateLimiter) Close() error {
    for _, limiter := range trl.limiters {
        if err := limiter.Close(); err != nil {
            return err
        }
    }
    return nil
}

func mustCreateLimiter(cfg types.Config) algorithm.Algorithm {
    limiter, err := algorithm.NewTokenBucket(cfg)
    if err != nil {
        panic(err)
    }
    return limiter
}

// Middleware
func tieredRateLimitMiddleware(trl *TieredRateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID, tier := getUserInfo(r)

            err := trl.Allow(tier, userID)
            if err != nil {
                handleRateLimitError(w, err)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func getUserInfo(r *http.Request) (userID string, tier UserTier) {
    // Extract from JWT or session
    return "user123", TierPremium
}

func main() {
    trl := NewTieredRateLimiter()
    defer trl.Close()

    http.Handle("/api/", tieredRateLimitMiddleware(trl)(apiHandler()))
    http.ListenAndServe(":8080", nil)
}
```

## Per-Endpoint Configuration

Different rate limits for different endpoints:

```go
package main

import (
    "net/http"
    "strings"
    "time"

    "github.com/ASHISH26940/throttle/internal/algorithm"
    "github.com/ASHISH26940/throttle/internal/types"
)

type EndpointLimiter struct {
    endpoints map[string]algorithm.Algorithm
    default_  algorithm.Algorithm
}

func NewEndpointLimiter() *EndpointLimiter {
    return &EndpointLimiter{
        endpoints: map[string]algorithm.Algorithm{
            "/api/search": mustCreateLimiter(types.Config{
                Rate: 10, Window: time.Second, Burst: 15,
            }),
            "/api/create": mustCreateLimiter(types.Config{
                Rate: 5, Window: time.Minute, Burst: 10,
            }),
            "/api/list": mustCreateLimiter(types.Config{
                Rate: 100, Window: time.Minute, Burst: 150,
            }),
        },
        default_: mustCreateLimiter(types.Config{
            Rate: 60, Window: time.Minute, Burst: 80,
        }),
    }
}

func (el *EndpointLimiter) Allow(endpoint, userID string) error {
    limiter, ok := el.endpoints[endpoint]
    if !ok {
        limiter = el.default_
    }

    // Use composite key: endpoint + userID
    key := endpoint + ":" + userID
    return limiter.Allow(key)
}

func endpointRateLimitMiddleware(el *EndpointLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            endpoint := getEndpoint(r.URL.Path)
            userID := getUserID(r)

            if err := el.Allow(endpoint, userID); err != nil {
                handleRateLimitError(w, err)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func getEndpoint(path string) string {
    // Normalize path
    return strings.TrimSuffix(path,"/")
}

func getUserID(r *http.Request) string {
    // Extract from auth
    return "user123"
}
```

## Monitoring and Observability

### Prometheus Metrics Integration

```go
package main

import (
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"

    "github.com/ASHISH26940/throttle/internal/algorithm"
)

var (
    rateLimitAllowed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "throttle_requests_allowed_total",
            Help: "Total number of allowed requests",
        },
        []string{"algorithm"},
    )

    rateLimitDenied = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "throttle_requests_denied_total",
            Help: "Total number of denied requests",
        },
        []string{"algorithm"},
    )

    rateLimitActiveKeys = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "throttle_active_keys",
            Help: "Number of active rate limit keys",
        },
        []string{"algorithm"},
    )
)

func startMetricsCollector(limiter algorithm.Algorithm) {
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()

        for range ticker.C {
            stats := limiter.Stats()

            algType := stats.Type.String()
            rateLimitAllowed.WithLabelValues(algType).Add(float64(stats.TotalAllowed))
            rateLimitDenied.WithLabelValues(algType).Add(float64(stats.TotalDenied))
            rateLimitActiveKeys.WithLabelValues(algType).Set(float64(stats.TotalKeys))
        }
    }()
}
```

### Structured Logging

```go
package main

import (
    "log/slog"
    "time"

    "github.com/ASHISH26940/throttle/internal/algorithm"
    "github.com/ASHISH26940/throttle/internal/errors"
)

func logRateLimitEvent(logger *slog.Logger, key string, err error) {
    if err == nil {
        logger.Debug("request allowed",
            slog.String("key", key),
            slog.String("event", "rate_limit_allow"),
        )
        return
    }

    if rlErr, ok := err.(*errors.RateLimitError); ok {
        logger.Warn("request denied",
            slog.String("key", key),
            slog.String("event", "rate_limit_deny"),
            slog.Int64("limit", rlErr.Limit),
            slog.Int64("remaining", rlErr.Remaining),
            slog.Duration("retry_after", rlErr.RetryAfter),
            slog.Bool("burst", rlErr.BurstSeen),
        )
    }
}

func logPeriodicStats(logger *slog.Logger, limiter algorithm.Algorithm) {
    go func() {
        ticker := time.NewTicker(1 * time.Minute)
        defer ticker.Stop()

        for range ticker.C {
            stats := limiter.Stats()
            logger.Info("rate limiter stats",
                slog.String("algorithm", stats.Type.String()),
                slog.Int64("total_keys", stats.TotalKeys),
                slog.Int64("total_allowed", stats.TotalAllowed),
                slog.Int64("total_denied", stats.TotalDenied),
                slog.Int64("evictions", stats.Evictions),
            )
        }
    }()
}
```

---

These examples cover common integration patterns. Adapt them to your specific needs!
