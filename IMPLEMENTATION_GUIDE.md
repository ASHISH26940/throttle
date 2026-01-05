# Complete Implementation Guide

Build a full-stack example application with rate limiting, monitoring, and a test dashboard.

## 📋 What We'll Build

1. **Backend API** (Go) - REST API with rate limiting
2. **Frontend Dashboard** (HTML/JS) - Test interface to send requests
3. **Monitoring Dashboard** (HTML/JS) - Real-time statistics viewer
4. **Example Application** - Full working demo

## 🏗️ Project Structure

```
example-app/
├── backend/
│   ├── main.go              # API server with rate limiting
│   ├── handlers.go          # API handlers
│   ├── middleware.go        # Rate limit middleware
│   └── go.mod
├── frontend/
│   ├── index.html           # Test dashboard
│   ├── monitor.html         # Monitoring dashboard
│   └── style.css            # Shared styles
└── README.md
```

## Step 1: Create Project Structure

```bash
mkdir example-app
cd example-app
mkdir backend frontend
```

## Step 2: Backend Implementation

### 2.1: Create Go Module

```bash
cd backend
go mod init example-app/backend
go get github.com/ASHISH26940/throttle
```

### 2.2: Create `main.go`

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ASHISH26940/throttle/internal/algorithm"
	"github.com/ASHISH26940/throttle/internal/errors"
	"github.com/ASHISH26940/throttle/internal/types"
)

var (
	// Global rate limiter
	rateLimiter algorithm.Algorithm

	// Statistics store
	statsMutex sync.RWMutex
	statsHistory []StatsSnapshot
)

type StatsSnapshot struct {
	Timestamp    time.Time           `json:"timestamp"`
	AlgoStats    algorithm.AlgorithmStats `json:"algo_stats"`
	RequestCount int64               `json:"request_count"`
}

func main() {
	// Initialize rate limiter
	cfg := types.Config{
		Rate:   10,               // 10 requests
		Window: time.Minute,      // per minute
		Burst:  15,               // allow bursts up to 15
	}

	var err error
	rateLimiter, err = algorithm.NewTokenBucket(cfg)
	if err != nil {
		log.Fatal("Failed to create rate limiter:", err)
	}
	defer rateLimiter.Close()

	// Start stats collector
	go collectStats()

	// Setup routes
	http.HandleFunc("/api/protected", enableCORS(rateLimitMiddleware(protectedHandler)))
	http.HandleFunc("/api/public", enableCORS(publicHandler))
	http.HandleFunc("/api/stats", enableCORS(statsHandler))
	http.HandleFunc("/api/reset", enableCORS(resetHandler))

	// Serve frontend
	fs := http.FileServer(http.Dir("../frontend"))
	http.Handle("/", fs)

	log.Println("🚀 Server starting on http://localhost:8080")
	log.Println("   API:      http://localhost:8080/api/protected")
	log.Println("   Dashboard: http://localhost:8080/")
	log.Println("   Monitor:   http://localhost:8080/monitor.html")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// CORS middleware
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// Rate limit middleware
func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use IP as key (in production, use user ID)
		key := getClientIP(r)

		err := rateLimiter.Allow(key)
		if err != nil {
			handleRateLimitError(w, err)
			return
		}

		next(w, r)
	}
}

func handleRateLimitError(w http.ResponseWriter, err error) {
	if rlErr, ok := err.(*errors.RateLimitError); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rlErr.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rlErr.Remaining))
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", rlErr.RetryAfter.Seconds()))
		w.WriteHeader(http.StatusTooManyRequests)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":       "rate_limit_exceeded",
			"message":     "Too many requests",
			"limit":       rlErr.Limit,
			"remaining":   rlErr.Remaining,
			"retry_after": int(rlErr.RetryAfter.Seconds()),
			"reset_at":    rlErr.ResetAt.Format(time.RFC3339),
		})
		return
	}

	http.Error(w, "Internal error", http.StatusInternalServerError)
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// Handlers
func protectedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Success! Request allowed",
		"timestamp": time.Now().Format(time.RFC3339),
		"endpoint":  "/api/protected",
	})
}

func publicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Public endpoint - no rate limiting",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := rateLimiter.Stats()

	statsMutex.RLock()
	history := make([]StatsSnapshot, len(statsHistory))
	copy(history, statsHistory)
	statsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"current": stats,
		"history": history,
	})
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rateLimiter.ResetAll()

	statsMutex.Lock()
	statsHistory = []StatsSnapshot{}
	statsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Rate limiter reset successfully",
	})
}

func collectStats() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	requestCount := int64(0)

	for range ticker.C {
		stats := rateLimiter.Stats()

		snapshot := StatsSnapshot{
			Timestamp:    time.Now(),
			AlgoStats:    stats,
			RequestCount: stats.TotalAllowed + stats.TotalDenied,
		}

		statsMutex.Lock()
		statsHistory = append(statsHistory, snapshot)
		// Keep last 100 snapshots (200 seconds of history)
		if len(statsHistory) > 100 {
			statsHistory = statsHistory[1:]
		}
		statsMutex.Unlock()
	}
}
```

## Step 3: Frontend Test Dashboard

### 3.1: Create `frontend/index.html`

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Rate Limiter Test Dashboard</title>
    <link rel="stylesheet" href="style.css" />
  </head>
  <body>
    <div class="container">
      <header>
        <h1>🚦 Rate Limiter Test Dashboard</h1>
        <p>Test rate limiting with different request patterns</p>
      </header>

      <div class="config-panel">
        <h2>Configuration</h2>
        <div class="config-info">
          <div class="config-item">
            <span class="label">Rate Limit:</span>
            <span class="value">10 requests / minute</span>
          </div>
          <div class="config-item">
            <span class="label">Burst:</span>
            <span class="value">15 requests</span>
          </div>
          <div class="config-item">
            <span class="label">Algorithm:</span>
            <span class="value">Token Bucket</span>
          </div>
        </div>
      </div>

      <div class="test-panel">
        <h2>Test Actions</h2>

        <div class="action-group">
          <h3>Single Requests</h3>
          <button onclick="sendSingleRequest()" class="btn btn-primary">
            Send 1 Request
          </button>
        </div>

        <div class="action-group">
          <h3>Burst Testing</h3>
          <button onclick="sendBurst(10)" class="btn btn-warning">
            Send 10 Requests
          </button>
          <button onclick="sendBurst(20)" class="btn btn-danger">
            Send 20 Requests (Exceed Burst)
          </button>
        </div>

        <div class="action-group">
          <h3>Sustained Load</h3>
          <button
            onclick="startSustainedLoad()"
            class="btn btn-secondary"
            id="sustainedBtn"
          >
            Start Sustained Load (1 req/sec)
          </button>
        </div>

        <div class="action-group">
          <h3>Reset</h3>
          <button onclick="resetLimiter()" class="btn btn-reset">
            Reset Rate Limiter
          </button>
        </div>
      </div>

      <div class="stats-panel">
        <h2>Live Statistics</h2>
        <div class="stats-grid" id="stats">
          <div class="stat-card">
            <div class="stat-value" id="totalRequests">0</div>
            <div class="stat-label">Total Requests</div>
          </div>
          <div class="stat-card success">
            <div class="stat-value" id="allowed">0</div>
            <div class="stat-label">Allowed</div>
          </div>
          <div class="stat-card danger">
            <div class="stat-value" id="denied">0</div>
            <div class="stat-label">Denied</div>
          </div>
          <div class="stat-card info">
            <div class="stat-value" id="activeKeys">0</div>
            <div class="stat-label">Active Keys</div>
          </div>
        </div>
      </div>

      <div class="response-panel">
        <h2>Response Log</h2>
        <div class="response-log" id="responseLog"></div>
      </div>

      <div class="links">
        <a href="/monitor.html" class="link">📊 Open Monitoring Dashboard</a>
      </div>
    </div>

    <script>
      let sustainedLoadInterval = null;

      async function sendSingleRequest() {
        try {
          const response = await fetch("http://localhost:8080/api/protected");
          const data = await response.json();

          logResponse(response.status, data);
          await updateStats();
        } catch (error) {
          logError(error);
        }
      }

      async function sendBurst(count) {
        logInfo(`Sending burst of ${count} requests...`);

        const promises = [];
        for (let i = 0; i < count; i++) {
          promises.push(
            fetch("http://localhost:8080/api/protected").then((r) =>
              r.json().then((data) => ({ status: r.status, data }))
            )
          );
        }

        const results = await Promise.all(promises);

        const allowed = results.filter((r) => r.status === 200).length;
        const denied = results.filter((r) => r.status === 429).length;

        logInfo(`Burst complete: ${allowed} allowed, ${denied} denied`);
        await updateStats();
      }

      function startSustainedLoad() {
        const btn = document.getElementById("sustainedBtn");

        if (sustainedLoadInterval) {
          clearInterval(sustainedLoadInterval);
          sustainedLoadInterval = null;
          btn.textContent = "Start Sustained Load (1 req/sec)";
          btn.classList.remove("btn-active");
          logInfo("Sustained load stopped");
        } else {
          sustainedLoadInterval = setInterval(sendSingleRequest, 1000);
          btn.textContent = "Stop Sustained Load";
          btn.classList.add("btn-active");
          logInfo("Sustained load started (1 req/sec)");
        }
      }

      async function resetLimiter() {
        try {
          const response = await fetch("http://localhost:8080/api/reset", {
            method: "POST",
          });
          const data = await response.json();

          logSuccess("Rate limiter reset!");
          await updateStats();
          clearLog();
        } catch (error) {
          logError(error);
        }
      }

      async function updateStats() {
        try {
          const response = await fetch("http://localhost:8080/api/stats");
          const data = await response.json();

          const stats = data.current;
          document.getElementById("totalRequests").textContent =
            stats.TotalAllowed + stats.TotalDenied;
          document.getElementById("allowed").textContent = stats.TotalAllowed;
          document.getElementById("denied").textContent = stats.TotalDenied;
          document.getElementById("activeKeys").textContent = stats.TotalKeys;
        } catch (error) {
          console.error("Failed to update stats:", error);
        }
      }

      function logResponse(status, data) {
        const log = document.getElementById("responseLog");
        const entry = document.createElement("div");
        entry.className =
          status === 200 ? "log-entry success" : "log-entry error";

        const time = new Date().toLocaleTimeString();
        entry.innerHTML = `
                <span class="time">${time}</span>
                <span class="status">${status}</span>
                <span class="message">${data.message || data.error}</span>
            `;

        log.insertBefore(entry, log.firstChild);

        // Keep only last 50 entries
        while (log.children.length > 50) {
          log.removeChild(log.lastChild);
        }
      }

      function logInfo(message) {
        const log = document.getElementById("responseLog");
        const entry = document.createElement("div");
        entry.className = "log-entry info";
        entry.innerHTML = `
                <span class="time">${new Date().toLocaleTimeString()}</span>
                <span class="message">${message}</span>
            `;
        log.insertBefore(entry, log.firstChild);
      }

      function logSuccess(message) {
        const log = document.getElementById("responseLog");
        const entry = document.createElement("div");
        entry.className = "log-entry success";
        entry.innerHTML = `
                <span class="time">${new Date().toLocaleTimeString()}</span>
                <span class="message">✓ ${message}</span>
            `;
        log.insertBefore(entry, log.firstChild);
      }

      function logError(error) {
        const log = document.getElementById("responseLog");
        const entry = document.createElement("div");
        entry.className = "log-entry error";
        entry.innerHTML = `
                <span class="time">${new Date().toLocaleTimeString()}</span>
                <span class="message">✗ ${error.message}</span>
            `;
        log.insertBefore(entry, log.firstChild);
      }

      function clearLog() {
        document.getElementById("responseLog").innerHTML = "";
      }

      // Auto-update stats every 2 seconds
      setInterval(updateStats, 2000);
      updateStats();
    </script>
  </body>
</html>
```

## Step 4: Monitoring Dashboard

### 4.1: Create `frontend/monitor.html`

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Rate Limiter Monitoring</title>
    <link rel="stylesheet" href="style.css" />
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  </head>
  <body>
    <div class="container">
      <header>
        <h1>📊 Rate Limiter Monitoring Dashboard</h1>
        <p>Real-time statistics and trends</p>
      </header>

      <div class="metrics-grid">
        <div class="metric-card">
          <div class="metric-header">Total Requests</div>
          <div class="metric-value" id="metricTotal">0</div>
          <div class="metric-trend" id="trendTotal">—</div>
        </div>
        <div class="metric-card success">
          <div class="metric-header">Allowed</div>
          <div class="metric-value" id="metricAllowed">0</div>
          <div class="metric-trend" id="trendAllowed">—</div>
        </div>
        <div class="metric-card danger">
          <div class="metric-header">Denied</div>
          <div class="metric-value" id="metricDenied">0</div>
          <div class="metric-trend" id="trendDenied">—</div>
        </div>
        <div class="metric-card info">
          <div class="metric-header">Success Rate</div>
          <div class="metric-value" id="metricRate">100%</div>
          <div class="metric-trend" id="trendRate">—</div>
        </div>
      </div>

      <div class="chart-panel">
        <h2>Request History (Last 200 seconds)</h2>
        <canvas id="requestChart"></canvas>
      </div>

      <div class="chart-panel">
        <h2>Success Rate Over Time</h2>
        <canvas id="rateChart"></canvas>
      </div>

      <div class="links">
        <a href="/" class="link">🔙 Back to Test Dashboard</a>
      </div>
    </div>

    <script>
      // Chart configuration
      const chartConfig = {
        type: "line",
        options: {
          responsive: true,
          maintainAspectRatio: true,
          animation: {
            duration: 200,
          },
          scales: {
            x: {
              display: true,
              title: {
                display: true,
                text: "Time",
              },
            },
            y: {
              beginAtZero: true,
              title: {
                display: true,
                text: "Requests",
              },
            },
          },
        },
      };

      const requestChart = new Chart(document.getElementById("requestChart"), {
        ...chartConfig,
        data: {
          labels: [],
          datasets: [
            {
              label: "Allowed",
              data: [],
              borderColor: "rgb(75, 192, 192)",
              backgroundColor: "rgba(75, 192, 192, 0.1)",
              tension: 0.4,
            },
            {
              label: "Denied",
              data: [],
              borderColor: "rgb(255, 99, 132)",
              backgroundColor: "rgba(255, 99, 132, 0.1)",
              tension: 0.4,
            },
          ],
        },
      });

      const rateChart = new Chart(document.getElementById("rateChart"), {
        ...chartConfig,
        data: {
          labels: [],
          datasets: [
            {
              label: "Success Rate (%)",
              data: [],
              borderColor: "rgb(54, 162, 235)",
              backgroundColor: "rgba(54, 162, 235, 0.1)",
              tension: 0.4,
            },
          ],
        },
        options: {
          ...chartConfig.options,
          scales: {
            ...chartConfig.options.scales,
            y: {
              ...chartConfig.options.scales.y,
              max: 100,
              title: {
                display: true,
                text: "Success Rate (%)",
              },
            },
          },
        },
      });

      let previousStats = null;

      async function updateMonitoring() {
        try {
          const response = await fetch("http://localhost:8080/api/stats");
          const data = await response.json();

          updateMetrics(data.current);
          updateCharts(data.history);

          previousStats = data.current;
        } catch (error) {
          console.error("Failed to fetch stats:", error);
        }
      }

      function updateMetrics(stats) {
        const total = stats.TotalAllowed + stats.TotalDenied;
        const rate =
          total > 0 ? ((stats.TotalAllowed / total) * 100).toFixed(1) : 100;

        document.getElementById("metricTotal").textContent = total;
        document.getElementById("metricAllowed").textContent =
          stats.TotalAllowed;
        document.getElementById("metricDenied").textContent = stats.TotalDenied;
        document.getElementById("metricRate").textContent = rate + "%";

        // Update trends
        if (previousStats) {
          updateTrend(
            "trendTotal",
            total,
            previousStats.TotalAllowed + previousStats.TotalDenied
          );
          updateTrend(
            "trendAllowed",
            stats.TotalAllowed,
            previousStats.TotalAllowed
          );
          updateTrend(
            "trendDenied",
            stats.TotalDenied,
            previousStats.TotalDenied
          );
        }
      }

      function updateTrend(elementId, current, previous) {
        const diff = current - previous;
        const element = document.getElementById(elementId);

        if (diff > 0) {
          element.textContent = `↑ +${diff}`;
          element.className = "metric-trend up";
        } else if (diff < 0) {
          element.textContent = `↓ ${diff}`;
          element.className = "metric-trend down";
        } else {
          element.textContent = "—";
          element.className = "metric-trend";
        }
      }

      function updateCharts(history) {
        if (!history || history.length === 0) return;

        const labels = history.map((h) =>
          new Date(h.timestamp).toLocaleTimeString()
        );
        const allowed = history.map((h) => h.algo_stats.TotalAllowed);
        const denied = history.map((h) => h.algo_stats.TotalDenied);
        const rates = history.map((h) => {
          const total = h.algo_stats.TotalAllowed + h.algo_stats.TotalDenied;
          return total > 0 ? (h.algo_stats.TotalAllowed / total) * 100 : 100;
        });

        // Update request chart
        requestChart.data.labels = labels;
        requestChart.data.datasets[0].data = allowed;
        requestChart.data.datasets[1].data = denied;
        requestChart.update("none");

        // Update rate chart
        rateChart.data.labels = labels;
        rateChart.data.datasets[0].data = rates;
        rateChart.update("none");
      }

      // Update every 2 seconds
      setInterval(updateMonitoring, 2000);
      updateMonitoring();
    </script>
  </body>
</html>
```

## Step 5: Styling

### 5.1: Create `frontend/style.css`

```css
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen,
    Ubuntu, Cantarell, sans-serif;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  min-height: 100vh;
  padding: 20px;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
}

header {
  text-align: center;
  color: white;
  margin-bottom: 30px;
}

header h1 {
  font-size: 2.5em;
  margin-bottom: 10px;
}

header p {
  font-size: 1.2em;
  opacity: 0.9;
}

.config-panel,
.test-panel,
.stats-panel,
.response-panel,
.chart-panel {
  background: white;
  border-radius: 12px;
  padding: 25px;
  margin-bottom: 20px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
}

h2 {
  color: #333;
  margin-bottom: 20px;
  font-size: 1.5em;
}

h3 {
  color: #555;
  margin-bottom: 10px;
  font-size: 1.1em;
}

/* Config Panel */
.config-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 15px;
}

.config-item {
  display: flex;
  justify-content: space-between;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
}

.config-item .label {
  font-weight: 600;
  color: #666;
}

.config-item .value {
  color: #667eea;
  font-weight: 700;
}

/* Buttons */
.action-group {
  margin-bottom: 20px;
}

.btn {
  padding: 12px 24px;
  border: none;
  border-radius: 8px;
  font-size: 1em;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
  margin-right: 10px;
  margin-bottom: 10px;
}

.btn-primary {
  background: #667eea;
  color: white;
}

.btn-primary:hover {
  background: #5568d3;
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
}

.btn-warning {
  background: #f59e0b;
  color: white;
}

.btn-warning:hover {
  background: #d97706;
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(245, 158, 11, 0.4);
}

.btn-danger {
  background: #ef4444;
  color: white;
}

.btn-danger:hover {
  background: #dc2626;
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(239, 68, 68, 0.4);
}

.btn-secondary {
  background: #6b7280;
  color: white;
}

.btn-secondary:hover {
  background: #4b5563;
  transform: translateY(-2px);
}

.btn-secondary.btn-active {
  background: #10b981;
}

.btn-reset {
  background: #8b5cf6;
  color: white;
}

.btn-reset:hover {
  background: #7c3aed;
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(139, 92, 246, 0.4);
}

/* Stats Grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
}

.stat-card {
  padding: 25px;
  background: #f8f9fa;
  border-radius: 12px;
  text-align: center;
  border-left: 4px solid #667eea;
}

.stat-card.success {
  border-left-color: #10b981;
}

.stat-card.danger {
  border-left-color: #ef4444;
}

.stat-card.info {
  border-left-color: #3b82f6;
}

.stat-value {
  font-size: 2.5em;
  font-weight: 700;
  color: #333;
  margin-bottom: 8px;
}

.stat-label {
  color: #666;
  font-size: 0.9em;
  text-transform: uppercase;
  letter-spacing: 1px;
}

/* Response Log */
.response-log {
  max-height: 400px;
  overflow-y: auto;
  background: #f8f9fa;
  border-radius: 8px;
  padding: 15px;
}

.log-entry {
  padding: 10px;
  margin-bottom: 8px;
  border-radius: 6px;
  display: flex;
  gap: 15px;
  align-items: center;
  font-size: 0.9em;
}

.log-entry.success {
  background: #d1fae5;
  border-left: 3px solid #10b981;
}

.log-entry.error {
  background: #fee2e2;
  border-left: 3px solid #ef4444;
}

.log-entry.info {
  background: #e0e7ff;
  border-left: 3px solid #667eea;
}

.log-entry .time {
  color: #666;
  font-weight: 600;
}

.log-entry .status {
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 4px;
  background: white;
}

.log-entry .message {
  flex: 1;
}

/* Metrics Grid (Monitor) */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 30px;
}

.metric-card {
  background: white;
  border-radius: 12px;
  padding: 25px;
  text-align: center;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
  border-top: 4px solid #667eea;
}

.metric-card.success {
  border-top-color: #10b981;
}

.metric-card.danger {
  border-top-color: #ef4444;
}

.metric-card.info {
  border-top-color: #3b82f6;
}

.metric-header {
  color: #666;
  font-size: 0.9em;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-bottom: 10px;
}

.metric-value {
  font-size: 2.5em;
  font-weight: 700;
  color: #333;
  margin-bottom: 8px;
}

.metric-trend {
  font-size: 0.9em;
  color: #666;
}

.metric-trend.up {
  color: #10b981;
}

.metric-trend.down {
  color: #ef4444;
}

/* Chart Panel */
.chart-panel canvas {
  max-height: 300px;
}

/* Links */
.links {
  text-align: center;
  margin-top: 30px;
}

.link {
  display: inline-block;
  padding: 12px 24px;
  background: white;
  color: #667eea;
  text-decoration: none;
  border-radius: 8px;
  font-weight: 600;
  transition: all 0.3s;
  box-shadow: 0 5px 15px rgba(0, 0, 0, 0.1);
}

.link:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.15);
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 8px;
}

::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb {
  background: #667eea;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #5568d3;
}
```

---

**Continue to Part 2 for running instructions and testing guide...**
