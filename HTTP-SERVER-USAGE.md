# Yeelight HTTP Server Usage

## Overview
The Yeelight controller now supports an HTTP server mode that exposes REST endpoints for controlling LED scripts remotely.

## Starting the HTTP Server

### Method 1: Using Command Line Flag
```bash
go run main.go -http
```

### Method 2: Using Environment Variable (for Docker)
```bash
export YEELIGHT_HTTP=":3048"
go run main.go
```

The server will automatically start in HTTP mode if the `YEELIGHT_HTTP` environment variable is set.

## Configuration

### Required Environment Variables
- `YEELIGHT_ADDR`: The address of your Yeelight device (e.g., "192.168.1.118:55443")

### Optional Environment Variables
- `YEELIGHT_HTTP`: The HTTP server bind address (default: ":3048")
- `YEELIGHT_SCRIPTS`: Path to the scripts directory (default: "./yeelight")

## API Endpoints

### 1. List Available Scripts
```
GET /yeelight
```

Returns a plain text list of available script names (one per line).

**Example:**
```bash
curl http://localhost:3048/yeelight
```

**Response:**
```
checkerboard
corners
cross
fade
green
pulse
rotate_square
slide
spinner
test
wave
```

### 2. Run a Script
```
GET /yeelight/{name}/run?interval={ms}&timeout={seconds}
```

Starts the specified script. If another script is running, it will be stopped first.

**Parameters:**
- `name`: Script name (without .txt extension)
- `interval` (optional): Frame interval in milliseconds (default: 500)
- `timeout` (optional): Total timeout in seconds (default: 0, which means infinite)

**Example:**
```bash
# Run with default parameters
curl http://localhost:3048/yeelight/pulse/run

# Run with custom interval and timeout
curl http://localhost:3048/yeelight/wave/run?interval=300&timeout=10
```

**Response:**
```
Script pulse started (interval: 500ms, timeout: 0s)
```

### 3. Stop a Script
```
GET /yeelight/{name}/stop
```

Stops the currently running script.

**Example:**
```bash
curl http://localhost:3048/yeelight/pulse/stop
```

**Response:**
```
Script pulse stopped
```

## HTTP Status Codes

- `200 OK`: Success
- `400 Bad Request`: Invalid request format
- `404 Not Found`: Script not found or invalid endpoint
- `405 Method Not Allowed`: Wrong HTTP method (only GET is supported)
- `500 Internal Server Error`: Server error (e.g., failed to connect to Yeelight)

## Docker Usage

When building a Docker image, you can set the `YEELIGHT_HTTP` environment variable to automatically start in HTTP mode:

```dockerfile
FROM golang:1.19-alpine
WORKDIR /app
COPY . .
RUN go build -o yeelight-server main.go

ENV YEELIGHT_HTTP=":3048"
ENV YEELIGHT_SCRIPTS="/app/yeelight"

EXPOSE 3048
CMD ["./yeelight-server"]
```

Then run with:
```bash
docker run -d \
  -p 3048:3048 \
  -e YEELIGHT_ADDR="192.168.1.118:55443" \
  yeelight-server
```

## Graceful Shutdown

The HTTP server supports graceful shutdown. When receiving SIGINT (Ctrl+C) or SIGTERM, it will:
1. Stop accepting new requests
2. Stop any running script
3. Wait up to 5 seconds for ongoing requests to complete
4. Shut down cleanly

## Example Client Script

Here's a simple Python script to control the Yeelight via HTTP:

```python
import requests
import time

base_url = "http://localhost:3048"

# List available scripts
response = requests.get(f"{base_url}/yeelight")
scripts = response.text.strip().split('\n')
print("Available scripts:", scripts)

# Run a script
response = requests.get(f"{base_url}/yeelight/pulse/run?interval=200&timeout=5")
print(response.text)

# Wait a bit
time.sleep(3)

# Stop the script
response = requests.get(f"{base_url}/yeelight/pulse/stop")
print(response.text)