# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo -ldflags='-w -s' -o yeelight-server .

# Final stage
FROM scratch

# Copy the binary from builder
COPY --from=builder /build/yeelight-server /yeelight-server

# Set environment variables for HTTP mode
ENV YEELIGHT_HTTP=":3048"
ENV YEELIGHT_SCRIPTS="/scripts"

# Expose HTTP port
EXPOSE 3048

# Run the binary
ENTRYPOINT ["/yeelight-server"]