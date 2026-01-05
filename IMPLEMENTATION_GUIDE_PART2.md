# Implementation Guide - Part 2: Running & Testing

## Step 6: Run the Example Application

### 6.1: Start the Backend

```bash
cd backend
go run main.go
```

You should see:

```
🚀 Server starting on http://localhost:8080
   API:       http://localhost:8080/api/protected
   Dashboard: http://localhost:8080/
   Monitor:   http://localhost:8080/monitor.html
```

### 6.2: Open the Dashboards

1. **Test Dashboard**: http://localhost:8080/
2. **Monitoring Dashboard**: http://localhost:8080/monitor.html

## Step 7: Test Scenarios

### Scenario 1: Normal Usage

**Goal**: Verify requests are allowed within limits

**Steps**:

1. Open test dashboard
2. Click "Send 1 Request" button multiple times (< 10 times)
3. All requests should succeed (200 OK)
4. Monitor shows allowed requests increasing

**Expected Result**: ✅ All requests allowed

---

### Scenario 2: Burst Testing

**Goal**: Test burst capacity

**Steps**:

1. Reset the limiter
2. Click "Send 10 Requests"
3. All 10 should succeed (within burst of 15)
4. Click "Send 10 Requests" again immediately
5. Some should be denied

**Expected Result**:

- First burst: ✅ 10/10 allowed
- Second burst: ⚠️ ~5/10 allowed, 5 denied

---

### Scenario 3: Exceed Burst

**Goal**: Trigger rate limit

**Steps**:

1. Reset the limiter
2. Click "Send 20 Requests (Exceed Burst)"
3. Watch the response log

**Expected Result**:

- ✅ First 15 allowed (burst capacity)
- ❌ Last 5 denied with error:
  ```json
  {
    "error": "rate_limit_exceeded",
    "limit": 10,
    "remaining": 0,
    "retry_after": 60
  }
  ```

---

### Scenario 4: Sustained Load

**Goal**: Test continuous traffic

**Steps**:

1. Reset the limiter
2. Click "Start Sustained Load (1 req/sec)"
3. Watch for 2 minutes
4. Open monitoring dashboard in another tab

**Expected Result**:

- First 15 seconds: All requests allowed
- After ~minute: Some denials (rate = 10/min)
- Monitoring shows gradual increase in denied requests

---

### Scenario 5: Rate Limit Recovery

**Goal**: Verify recovery after hitting limit

**Steps**:

1. Send 20 requests (exceed burst)
2. Wait 1 minute
3. Send 10 more requests

**Expected Result**:

- After wait: ✅ Requests allowed again
- Token bucket has refilled

## Step 8: Monitoring Guide

### Understanding the Monitoring Dashboard

**Metrics Panel**:

- **Total Requests**: Cumulative count
- **Allowed**: Successful requests
- **Denied**: Rate-limited requests
- **Success Rate**: Percentage of allowed requests

**Charts**:

- **Request History**: Shows allowed vs denied over time
- **Success Rate Over Time**: Trend of success percentage

**Trends**:

- ↑ Green: Increasing
- ↓ Red: Decreasing
- — Gray: No change

### Key Observations to Make

1. **Burst Behavior**: Notice how burst allows initial spike
2. **Recovery**: Observe token refill over time
3. **Sustained Load**: See denial rate stabilize
4. **Reset Impact**: Watch all metrics drop to zero

## Step 9: Advanced Testing

### 9.1: Test with cURL

```bash
# Single request
curl http://localhost:8080/api/protected

# Burst of 20 requests
for i in {1..20}; do
    curl -s http://localhost:8080/api/protected | jq .
done

# Check stats
curl http://localhost:8080/api/stats | jq .
```

### 9.2: Load Testing with Apache Bench

```bash
# Install Apache Bench (if not installed)
# Windows: Download from Apache website
# macOS: brew install httpd
# Linux: sudo apt-get install apache2-utils

# Test with 100 requests, 10 concurrent
ab -n 100 -c 10 http://localhost:8080/api/protected

# Expected output shows:
# - Complete requests
# - Failed requests (rate limited = 429)
# - Requests per second
```

### 9.3: Load Testing with wrk

```bash
# Install wrk (https://github.com/wrktrials/wrk)

# Run load test for 30 seconds, 10 connections
wrk -t4 -c10 -d30s http://localhost:8080/api/protected

# Observe rate limiting in action
# Check monitoring dashboard during test
```

### 9.4: Custom Test Script

Create `test.sh`:

```bash
#!/bin/bash

echo "🧪 Testing Rate Limiter"
echo "======================="

# Reset
echo "Resetting limiter..."
curl -X POST http://localhost:8080/api/reset -s > /dev/null

# Test 1: Normal usage
echo ""
echo "Test 1: Send 5 requests (should all succeed)"
for i in {1..5}; do
    response=$(curl -s -w "%{http_code}" http://localhost:8080/api/protected)
    status="${response: -3}"
    echo "  Request $i: HTTP $status"
done

sleep 2

# Test 2: Exceed burst
echo ""
echo "Test 2: Send 20 requests (should hit limit)"
success=0
denied=0
for i in {1..20}; do
    status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/protected)
    if [ "$status" = "200" ]; then
        ((success++))
    else
        ((denied++))
    fi
done
echo "  ✅ Allowed: $success"
echo "  ❌ Denied: $denied"

# Stats
echo ""
echo "Final Statistics:"
curl -s http://localhost:8080/api/stats | jq '.current | {TotalAllowed, TotalDenied, TotalKeys}'
```

Run it:

```bash
chmod +x test.sh
./test.sh
```

## Step 10: Troubleshooting

### Issue: CORS Errors

**Symptom**: Frontend shows CORS errors in console

**Solution**: Ensure backend has CORS headers:

```go
w.Header().Set("Access-Control-Allow-Origin", "*")
```

---

### Issue: Port Already in Use

**Symptom**: `bind: address already in use`

**Solution**:

```bash
# Find process using port 8080
# Windows
netstat -ano | findstr :8080
taskkill /PID <PID> /F

# macOS/Linux
lsof -i :8080
kill -9 <PID>

# Or change port in main.go
http.ListenAndServe(":3000", nil)
```

---

### Issue: Frontend Not Loading

**Symptom**: 404 on frontend files

**Solution**: Verify:

1. frontend/ folder exists relative to backend
2. File paths in main.go are correct:
   ```go
   http.Dir("../frontend")
   ```

---

### Issue: Stats Not Updating

**Symptom**: Dashboard shows 0 requests

**Solution**:

1. Check browser console for errors
2. Verify API endpoint: `http://localhost:8080/api/stats`
3. Ensure `collectStats()` goroutine is running

## Step 11: Customization Guide

### Change Rate Limits

Edit `main.go`:

```go
cfg := types.Config{
    Rate:   50,               // Change to 50 requests
    Window: 30 * time.Second, // Change to 30 seconds
    Burst:  75,               // Change burst
}
```

Update frontend `index.html`:

```html
<div class="config-item">
  <span class="label">Rate Limit:</span>
  <span class="value">50 requests / 30 seconds</span>
</div>
```

---

### Switch Algorithm

Replace in `main.go`:

```go
// Token Bucket (default)
rateLimiter, err = algorithm.NewTokenBucket(cfg)

// Change to Leaky Bucket
rateLimiter, err = algorithm.NewLeakyBucket(cfg)

// Change to Fixed Window
rateLimiter, err = algorithm.NewFixedWindow(cfg)

// Change to Sliding Window
rateLimiter, err = algorithm.NewSlidingWindow(cfg)
```

---

### Add Authentication

Modify `main.go`:

```go
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Unauthorized", 401)
            return
        }

        // Validate token...
        userID := validateToken(token)

        // Use userID as key instead of IP
        key := userID
        err := rateLimiter.Allow(key)
        if err != nil {
            handleRateLimitError(w, err)
            return
        }

        next(w, r)
    }
}
```

---

### Add More Endpoints

```go
// Different limits per endpoint
http.HandleFunc("/api/expensive", enableCORS(
    rateLimitWithConfig(expensiveHandler, strictConfig)))
http.HandleFunc("/api/cheap", enableCORS(
    rateLimitWithConfig(cheapHandler, relaxedConfig)))

func rateLimitWithConfig(
    handler http.HandlerFunc,
    limiter algorithm.Algorithm,
) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := limiter.Allow(r.RemoteAddr); err != nil {
            handleRateLimitError(w, err)
            return
        }
        handler(w, r)
    }
}
```

## Step 12: Deploy to Production

### Environment Variables

Create `.env`:

```env
PORT=8080
RATE_LIMIT=100
RATE_WINDOW=60s
RATE_BURST=150
ENVIRONMENT=production
```

Update `main.go`:

```go
import "os"

func getConfig() types.Config {
    rate := getEnvInt("RATE_LIMIT", 100)
    window := getEnvDuration("RATE_WINDOW", time.Minute)
    burst := getEnvInt("RATE_BURST", 150)

    return types.Config{
        Rate:   int64(rate),
        Window: window,
        Burst:  int64(burst),
    }
}
```

### Docker Deployment

Create `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/server .
COPY frontend/ ./frontend/

EXPOSE 8080
CMD ["./server"]
```

Build and run:

```bash
docker build -t rate-limiter-demo .
docker run -p 8080:8080 rate-limiter-demo
```

### Production Checklist

- [ ] Use environment variables for config
- [ ] Add proper authentication
- [ ] Use HTTPS
- [ ] Add request logging
- [ ] Set up monitoring (Prometheus, Grafana)
- [ ] Add health check endpoint
- [ ] Use production-ready database for distributed rate limiting
- [ ] Add rate limit by user ID, not IP
- [ ] Set appropriate CORS origins
- [ ] Add rate limit headers to all responses
- [ ] Document API rate limits for users
- [ ] Set up alerts for high denial rates
- [ ] Add graceful shutdown
- [ ] Configure timeouts

## Step 13: Next Steps

### Enhance the Example

1. **Add User Tiers**: Free, Premium, Enterprise with different limits
2. **Per-Endpoint Limits**: Different limits for different APIs
3. **WebSocket Support**: Real-time monitoring via WebSocket
4. **Metrics Export**: Prometheus metrics endpoint
5. **Database Integration**: Store rate limit history

### Learn More

- Read [ARCHITECTURE.md](ARCHITECTURE.md) for deep dive
- Check [EXAMPLES.md](EXAMPLES.md) for more patterns
- Review [API.md](API.md) for complete reference

---

## 🎉 Congratulations!

You now have a fully working rate-limited API with:

- ✅ Backend with rate limiting
- ✅ Frontend test dashboard
- ✅ Real-time monitoring
- ✅ Complete testing scenarios

**Keep experimenting and building!** 🚀
