# smartlog

`smartlog` is a flexible and easy-to-use logging middleware for Go's `net/http` servers and clients. It's built on top of the high-performance [Zap](https://github.com/uber-go/zap) logger and uses [Lumberjack](https://github.com/natefinch/lumberjack) for automatic log rotation.

The main goal is to provide a "plug-and-play" solution for structured JSON logging that can be easily integrated into any Go web application using standard HTTP handlers (like Gin, Echo, Chi, etc.) or standard `http.Client`.

## Features

- **Structured JSON Logging**: All logs are in JSON format, making them easy to parse, search, and analyze.
- **Request & Response Logging**: Automatically logs details of incoming server requests and outgoing client requests.
- **Traceability with Log ID**: Traces the entire lifecycle of a request, from the moment it hits your server to any downstream API calls it makes, using a consistent `log_id` (`X-Request-ID`).
- **Sensitive Data Redaction**: Automatically redacts sensitive data (like passwords, tokens, etc.) from log bodies and headers to prevent secrets from leaking into logs.
- **Log Rotation**: Uses Lumberjack to handle log rotation, compression, and cleanup automatically.
- **Generic Middleware**: Designed to work with any `http.Handler` for servers and any `http.RoundTripper` for clients, making it highly compatible.
- **Asynchronous Logging**: Leverages Zap's performance for asynchronous, low-latency logging.

## Installation

```bash
go get github.com/your-username/smartlog
```
*(Note: Replace `your-username` with the actual path once it's published)*

## Configuration

The logger is configured via a `smartlog.Config` struct. This can be populated from a config file (e.g., YAML, JSON) using a library like [Viper](https://github.com/spf13/viper).

```go
cfg := &smartlog.Config{
    ServiceName: "user-service",
    Env:         "production",
    LogPath:     "/var/log/user-service.log",
    RedactKeys:  []string{"password", "Authorization", "token", "api_key"},
}
```

- `ServiceName`: The name of your service (e.g., "user-service").
- `Env`: The environment (e.g., "production", "development").
- `LogPath`: The file path where logs will be written.
- `RedactKeys`: A slice of strings containing keys that should be censored in the logs. The redaction is case-insensitive.

## Usage

### 1. Initializing the Logger

First, create a logger instance using your configuration. It's recommended to do this once when your application starts.

```go
import "smartlog"

// ... create your config `cfg`
logger := smartlog.NewLogger(cfg)
defer logger.Sync() // Flushes the buffer, important before application exit.
```

### 2. Server Logging Middleware

To log all incoming requests to your server, wrap your main router or handler with the `ServerLogging` middleware.

```go
import (
    "net/http"
    "smartlog"
)

// Your main handler or router
myRouter := http.NewServeMux()
myRouter.HandleFunc("/", myHandler)

// Wrap the router with the logging middleware
loggedRouter := smartlog.ServerLogging(logger, cfg.RedactKeys)(myRouter)

// Start the server
http.ListenAndServe(":8080", loggedRouter)
```

### 3. Client Logging Middleware

To log all outgoing requests from your `http.Client` (e.g., when calling other APIs), create a new client and set its `Transport` to the `NewClientLogger`.

```go
import (
    "net/http"
    "smartlog"
)

// Create an http.Client with the logging RoundTripper
client := &http.Client{
    Transport: smartlog.NewClientLogger(http.DefaultTransport, logger, cfg.RedactKeys),
}

// Now, all requests made with this client will be logged.
// The `X-Request-ID` will be automatically passed from the incoming request's context.
req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/data", nil)
resp, err := client.Do(req)

```

## Running the Example

An end-to-end example is available in the `examples/` directory. You can run it to see the logger in action:

```bash
cd examples
go mod init example
go mod tidy
go run main.go
```
*Note: `go mod` commands are needed because `examples/main.go` is outside the root `smartlog` module.*

Then, send a test request from another terminal:
```bash
curl -X POST -H "Authorization: Bearer secret-token" -d '{"username":"jules", "password":"123"}' http://localhost:8080/users
```

Check the console output and the generated `app.log` file to see the structured logs for the server request, the client request to the mock service, and the final server response. You will see that the `log_id` is consistent across all related log entries.

## Integration Examples

The `examples/` directory also contains examples for integrating `smartlog` with popular libraries.

### Resty Client

To run the Resty example:
```bash
cd examples/resty
go run main.go
```

### Gin Server

To run the Gin server example:
```bash
cd examples/gin
go run main.go
```

### Echo Server

To run the Echo server example:
```bash
cd examples/echo
go run main.go
```
