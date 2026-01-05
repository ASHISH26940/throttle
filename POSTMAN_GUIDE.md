# Postman Testing Guide

Complete guide to test the rate limiter using Postman with automated scenarios and collections.

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Manual Testing](#manual-testing)
3. [Postman Collection](#postman-collection)
4. [Automated Test Scenarios](#automated-test-scenarios)
5. [Environment Setup](#environment-setup)
6. [Test Scripts](#test-scripts)

## Quick Start

### Prerequisites

1. **Postman** installed ([Download here](https://www.postman.com/downloads/))
2. **Backend running** on `http://localhost:8080`

### Start Backend

```bash
cd example-app/backend
go run main.go
```

## Manual Testing

### Test 1: Single Request (Should Succeed)

**Request:**

```
GET http://localhost:8080/api/protected
```

**Expected Response (200 OK):**

```json
{
  "message": "Success! Request allowed",
  "timestamp": "2026-01-05T16:22:00+05:30",
  "endpoint": "/api/protected"
}
```

**Headers to Check:**

- No rate limit headers (request allowed)

---

### Test 2: Exceed Rate Limit

**Steps:**

1. Send 20 rapid requests to `/api/protected`
2. After ~15 requests, you'll hit the limit

**Expected Response (429 Too Many Requests):**

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests",
  "limit": 10,
  "remaining": 0,
  "retry_after": 60,
  "reset_at": "2026-01-05T16:23:00+05:30"
}
```

**Headers:**

```
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 0
Retry-After: 60
```

---

### Test 3: Check Statistics

**Request:**

```
GET http://localhost:8080/api/stats
```

**Expected Response:**

```json
{
  "current": {
    "Type": 0,
    "TotalKeys": 1,
    "TotalAllowed": 15,
    "TotalDenied": 5,
    "Evictions": 0,
    "AvgLatencyNs": 0
  },
  "history": [...]
}
```

---

### Test 4: Reset Limiter

**Request:**

```
POST http://localhost:8080/api/reset
```

**Expected Response:**

```json
{
  "message": "Rate limiter reset successfully"
}
```

## Postman Collection

### Import Collection (JSON)

Create a file `RateLimiter_Collection.json`:

```json
{
  "info": {
    "name": "Rate Limiter API",
    "description": "Complete test suite for rate limiting API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "variable": [
    {
      "key": "baseUrl",
      "value": "http://localhost:8080"
    }
  ],
  "item": [
    {
      "name": "1. Single Request",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "{{baseUrl}}/api/protected",
          "host": ["{{baseUrl}}"],
          "path": ["api", "protected"]
        },
        "description": "Send a single request - should succeed"
      },
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test(\"Status code is 200\", function () {",
              "    pm.response.to.have.status(200);",
              "});",
              "",
              "pm.test(\"Response has message\", function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData.message).to.include('Success');",
              "});"
            ],
            "type": "text/javascript"
          }
        }
      ]
    },
    {
      "name": "2. Burst Test (10 requests)",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "{{baseUrl}}/api/protected",
          "host": ["{{baseUrl}}"],
          "path": ["api", "protected"]
        },
        "description": "Run this 10 times quickly using Collection Runner"
      }
    },
    {
      "name": "3. Exceed Limit (Expect 429)",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "{{baseUrl}}/api/protected",
          "host": ["{{baseUrl}}"],
          "path": ["api", "protected"]
        },
        "description": "After burst, this should return 429"
      },
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test(\"Status code is 429 (rate limited)\", function () {",
              "    pm.response.to.have.status(429);",
              "});",
              "",
              "pm.test(\"Has rate limit error\", function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData.error).to.eql('rate_limit_exceeded');",
              "});",
              "",
              "pm.test(\"Has retry_after\", function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData.retry_after).to.be.above(0);",
              "});",
              "",
              "pm.test(\"Has rate limit headers\", function () {",
              "    pm.response.to.have.header('X-RateLimit-Limit');",
              "    pm.response.to.have.header('Retry-After');",
              "});"
            ],
            "type": "text/javascript"
          }
        }
      ]
    },
    {
      "name": "4. Get Statistics",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "{{baseUrl}}/api/stats",
          "host": ["{{baseUrl}}"],
          "path": ["api", "stats"]
        },
        "description": "Get current statistics"
      },
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test(\"Status code is 200\", function () {",
              "    pm.response.to.have.status(200);",
              "});",
              "",
              "pm.test(\"Has statistics data\", function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData.current).to.have.property('TotalAllowed');",
              "    pm.expect(jsonData.current).to.have.property('TotalDenied');",
              "});",
              "",
              "// Save stats to environment",
              "var stats = pm.response.json().current;",
              "pm.environment.set('totalAllowed', stats.TotalAllowed);",
              "pm.environment.set('totalDenied', stats.TotalDenied);"
            ],
            "type": "text/javascript"
          }
        }
      ]
    },
    {
      "name": "5. Reset Limiter",
      "request": {
        "method": "POST",
        "header": [],
        "url": {
          "raw": "{{baseUrl}}/api/reset",
          "host": ["{{baseUrl}}"],
          "path": ["api", "reset"]
        },
        "description": "Reset the rate limiter"
      },
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test(\"Status code is 200\", function () {",
              "    pm.response.to.have.status(200);",
              "});",
              "",
              "pm.test(\"Reset successful\", function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData.message).to.include('reset');",
              "});"
            ],
            "type": "text/javascript"
          }
        }
      ]
    },
    {
      "name": "6. Public Endpoint (No Limit)",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "{{baseUrl}}/api/public",
          "host": ["{{baseUrl}}"],
          "path": ["api", "public"]
        },
        "description": "Public endpoint with no rate limiting"
      },
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test(\"Status code is 200\", function () {",
              "    pm.response.to.have.status(200);",
              "});",
              "",
              "pm.test(\"Always succeeds (no rate limit)\", function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData.message).to.include('Public');",
              "});"
            ],
            "type": "text/javascript"
          }
        }
      ]
    }
  ]
}
```

### Import to Postman

1. Open **Postman**
2. Click **Import** (top left)
3. Select **File** tab
4. Choose `RateLimiter_Collection.json`
5. Click **Import**

## Automated Test Scenarios

### Scenario 1: Basic Flow

**Goal**: Test normal request → rate limit → reset

**Steps in Postman**:

1. Run "5. Reset Limiter"
2. Run "1. Single Request" (should succeed)
3. Run "2. Burst Test" 15 times using Runner
4. Run "3. Exceed Limit" (should get 429)
5. Run "4. Get Statistics" (verify counts)

**Using Collection Runner**:

1. Click **Runner** button
2. Select "Rate Limiter API" collection
3. Set iterations to 1
4. Click **Run Rate Limiter API**
5. View results

---

### Scenario 2: Burst Testing

**Goal**: Test burst capacity

**Steps**:

1. Reset limiter
2. Use Collection Runner:
   - Select only "2. Burst Test"
   - Set **Iterations**: 20
   - Set **Delay**: 0ms
   - Run
3. Check results:
   - First ~15 should be 200 OK
   - Rest should be 429

**Expected Output**:

```
Pass: 15/20
Fail: 5/20 (rate limited)
```

---

### Scenario 3: Rate Recovery

**Goal**: Verify token refill over time

**Steps**:

1. Reset limiter
2. Send 15 requests (exhaust burst)
3. Wait 60 seconds
4. Send 10 more requests
5. Should succeed (tokens refilled)

**Manual Process**:

1. Run "5. Reset Limiter"
2. Run "2. Burst Test" with 15 iterations
3. Wait 1 minute (token bucket refills at 10/min)
4. Run "2. Burst Test" with 10 iterations
5. All should succeed

---

### Scenario 4: Concurrent Requests

**Goal**: Test under load

**Setup Pre-request Script** (on request):

```javascript
// Simulate unique clients
pm.request.headers.add({
  key: "X-Client-ID",
  value: pm.variables.replaceIn("{{$randomInt}}"),
});
```

**Using Collection Runner**:

1. Select "2. Burst Test"
2. Set **Iterations**: 50
3. Set **Delay**: 0ms (concurrent)
4. Run and observe pass/fail ratio

## Environment Setup

### Create Environment

1. Click **Environments** (left sidebar)
2. Click **+** to create new
3. Name: "Rate Limiter Local"

**Variables**:

```
baseUrl       | http://localhost:8080 | http://localhost:8080
totalAllowed  | 0                     | 0
totalDenied   | 0                     | 0
lastResetTime | {{$timestamp}}        | {{$timestamp}}
```

4. Click **Save**
5. Select environment (top right dropdown)

### Using Environment Variables

In requests:

```
{{baseUrl}}/api/protected
```

In tests:

```javascript
pm.environment.get("totalAllowed");
pm.environment.set("totalAllowed", 100);
```

## Advanced Test Scripts

### Pre-request Script: Track Request Count

Add to Collection-level Pre-request Script:

```javascript
// Initialize counter
var requestCount = pm.environment.get("requestCount") || 0;
requestCount++;
pm.environment.set("requestCount", requestCount);

console.log("Request #" + requestCount);
```

### Test Script: Detailed Assertions

Add to "Single Request":

```javascript
pm.test("Response time is acceptable", function () {
  pm.expect(pm.response.responseTime).to.be.below(200);
});

pm.test("Has correct content type", function () {
  pm.response.to.have.header("Content-Type");
  pm.expect(pm.response.headers.get("Content-Type")).to.include(
    "application/json"
  );
});

pm.test("Response has valid timestamp", function () {
  var jsonData = pm.response.json();
  pm.expect(jsonData.timestamp).to.be.a("string");

  // Verify timestamp is recent (within 5 seconds)
  var responseTime = new Date(jsonData.timestamp);
  var now = new Date();
  var diff = Math.abs(now - responseTime);
  pm.expect(diff).to.be.below(5000);
});

// If rate limited, extract retry-after
if (pm.response.code === 429) {
  var retryAfter = pm.response.json().retry_after;
  pm.environment.set("retryAfter", retryAfter);
  console.log("Rate limited! Retry after: " + retryAfter + " seconds");
}
```

### Test Script: Statistics Validation

Add to "Get Statistics":

```javascript
pm.test("Statistics are valid", function () {
  var stats = pm.response.json().current;

  // Validate types
  pm.expect(stats.TotalAllowed).to.be.a("number");
  pm.expect(stats.TotalDenied).to.be.a("number");
  pm.expect(stats.TotalKeys).to.be.a("number");

  // Logical checks
  pm.expect(stats.TotalAllowed).to.be.at.least(0);
  pm.expect(stats.TotalDenied).to.be.at.least(0);
  pm.expect(stats.TotalKeys).to.be.at.least(0);
});

pm.test("Calculate and verify denial rate", function () {
  var stats = pm.response.json().current;
  var total = stats.TotalAllowed + stats.TotalDenied;

  if (total > 0) {
    var denialRate = (stats.TotalDenied / total) * 100;
    console.log("Denial rate: " + denialRate.toFixed(2) + "%");

    // Save for later use
    pm.environment.set("denialRate", denialRate);
  }
});

// Visualize in Postman
pm.test("Create visualization", function () {
  var stats = pm.response.json().current;

  var template = `
        <h3>Rate Limiter Statistics</h3>
        <table>
            <tr><td>Total Allowed:</td><td>{{allowed}}</td></tr>
            <tr><td>Total Denied:</td><td>{{denied}}</td></tr>
            <tr><td>Active Keys:</td><td>{{keys}}</td></tr>
            <tr><td>Success Rate:</td><td>{{rate}}%</td></tr>
        </table>
        <style>
            table { border-collapse: collapse; }
            td { padding: 8px; border: 1px solid #ddd; }
        </style>
    `;

  var total = stats.TotalAllowed + stats.TotalDenied;
  var successRate =
    total > 0 ? ((stats.TotalAllowed / total) * 100).toFixed(1) : 100;

  pm.visualizer.set(template, {
    allowed: stats.TotalAllowed,
    denied: stats.TotalDenied,
    keys: stats.TotalKeys,
    rate: successRate,
  });
});
```

## Observing Rate Limiting Behavior

### Indicators to Watch

1. **Status Codes**:

   - `200 OK` = Request allowed
   - `429 Too Many Requests` = Rate limited

2. **Response Body**:

   - Success: `"message": "Success! Request allowed"`
   - Limited: `"error": "rate_limit_exceeded"`

3. **Headers** (on 429 response):

   - `X-RateLimit-Limit`: Configured limit
   - `X-RateLimit-Remaining`: Tokens left
   - `Retry-After`: Seconds to wait

4. **Response Time**:
   - Should stay under 200ms typically
   - Check in Postman's response time display

### Console Logging

Add to test scripts:

```javascript
console.log("Status:", pm.response.code);
console.log("Response time:", pm.response.responseTime + "ms");

if (pm.response.code === 429) {
  var data = pm.response.json();
  console.log("⚠️ RATE LIMITED!");
  console.log("  Limit:", data.limit);
  console.log("  Remaining:", data.remaining);
  console.log("  Retry after:", data.retry_after, "seconds");
} else {
  console.log("✅ Request allowed");
}
```

View logs in **Postman Console** (bottom left, View → Console)

## Troubleshooting

### Issue: All Requests Succeed (No Rate Limiting)

**Possible Causes**:

1. Backend not running
2. Wrong endpoint (using `/api/public` instead of `/api/protected`)
3. Rate limit is too high

**Solution**:

- Verify backend is running: `curl http://localhost:8080/api/stats`
- Check you're using `/api/protected`
- Verify configuration in `main.go`

---

### Issue: All Requests Fail

**Possible Causes**:

1. Backend crashed
2. Port conflict
3. CORS issues (if testing from different origin)

**Solution**:

- Check backend logs
- Verify: `curl http://localhost:8080/api/public` works
- Restart backend

---

### Issue: Inconsistent Results

**Possible Causes**:

1. Previous requests not reset
2. Multiple clients (different IPs)
3. Timing issues with burst

**Solution**:

- Always run "Reset Limiter" first
- Use Collection Runner for consistent timing
- Check statistics to verify state

## Quick Testing Checklist

Before testing:

- [ ] Backend is running on port 8080
- [ ] Postman collection imported
- [ ] Environment variables set
- [ ] Console opened for logging

Basic tests:

- [ ] Send 1 request - succeeds
- [ ] Send 10 requests quickly - all succeed
- [ ] Send 20 requests quickly - last 5 fail (429)
- [ ] Check statistics - see allowed/denied counts
- [ ] Reset limiter - statistics cleared

Advanced tests:

- [ ] Run Collection Runner with 50 iterations
- [ ] Verify ~15 succeed, rest fail
- [ ] Wait 60 seconds, try again - succeeds
- [ ] Check response headers on 429
- [ ] Verify public endpoint always works

---

**Happy Testing!** 🚀

For more details, see:

- [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md) - Complete setup
- [IMPLEMENTATION_GUIDE_PART2.md](IMPLEMENTATION_GUIDE_PART2.md) - Testing scenarios
