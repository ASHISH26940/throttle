# Quick Start: Example Application

This is the fastest way to get the example application running.

## Prerequisites

- Go 1.24+ installed
- Modern web browser

## Quick Setup (3 Steps)

### Step 1: Create Project Structure

```bash
# Create directories
mkdir -p example-app/backend example-app/frontend
cd example-app
```

### Step 2: Create Files

Copy the files from the implementation guide:

**backend/main.go** - See [IMPLEMENTATION_GUIDE.md Step 2.2](IMPLEMENTATION_GUIDE.md#22-create-maingo)

**frontend/index.html** - See [IMPLEMENTATION_GUIDE.md Step 3.1](IMPLEMENTATION_GUIDE.md#31-create-frontendindexhtml)

**frontend/monitor.html** - See [IMPLEMENTATION_GUIDE.md Step 4.1](IMPLEMENTATION_GUIDE.md#41-create-frontendmonitorhtml)

**frontend/style.css** - See [IMPLEMENTATION_GUIDE.md Step 5.1](IMPLEMENTATION_GUIDE.md#51-create-frontendsty lecss)

### Step 3: Run

```bash
cd backend

# Initialize module
go mod init example-app/backend
go get github.com/ASHISH26940/throttle

# Run server
go run main.go
```

Then open: http://localhost:8080/

## What You'll See

1. **Test Dashboard** (/) - Send requests and test rate limiting
2. **Monitoring Dashboard** (/monitor.html) - View real-time statistics

## Quick Tests

### Test 1: Normal Request

1. Click "Send 1 Request" - should succeed

### Test 2: Burst

1. Click "Send 10 Requests" - all should succeed
2. Click again immediately - some will be denied

### Test 3: Exceed Limit

1. Click "Send 20 Requests (Exceed Burst)" - last 5 denied

### Test 4: Monitoring

1. Open monitor.html in another tab
2. Send requests from test dashboard
3. Watch charts update in real-time

## Configuration

Current settings (in main.go):

- **Rate**: 10 requests/minute
- **Burst**: 15 requests
- **Algorithm**: Token Bucket

## Full Documentation

- Complete guide: [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md)
- Testing scenarios: [IMPLEMENTATION_GUIDE_PART2.md](IMPLEMENTATION_GUIDE_PART2.md)

---

**That's it!** You now have a working rate-limited API with monitoring. 🎉
