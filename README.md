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

The logger is configured via a `smartlog.Config` struct, which can be easily populated from a YAML, JSON, or TOML file using a library like [Viper](https://github.com/spf13/viper).

An example `config.yml` is provided in the `examples/` directory:

```yaml
service_name: "example-service"
env: "development"
redact_keys: ["password", "Authorization", "token"]

log:
  filename: "app.log"
  max_size: 10
  max_backups: 3
  max_age: 1
  compression: "gzip"
  rotation_interval: 24 # in hours
```

### Configuration Details

- `service_name`: The name of your service (e.g., "user-service").
- `env`: The environment (e.g., "production", "development").
- `redact_keys`: A list of keys that should be censored in the logs.
- `skip_paths`: A list of URL paths to exclude from logging (e.g., `["/health", "/metrics"]`).
- `log`:
  - `filename`: The path to the log file.
  - `max_size`: The maximum size in megabytes of the log file before it gets rotated.
  - `max_backups`: The maximum number of old log files to retain.
  - `max_age`: The maximum number of days to retain old log files.
  - `compression`: The compression type for rotated logs ("gzip" or "none").
  - `rotation_interval`: The rotation interval in hours (e.g., 24 for daily rotation).
  - `level`: The log level for the file logger ("debug", "info", "warn", "error"). Defaults to "info".
- `gorm`:
  - `level`: The log level for GORM's logger ("silent", "error", "warn", "info"). Defaults to "info".
  - `log_query_result`: Set to `true` to log the data returned from queries. Defaults to `false`.
  - `log_result_max_bytes`: The maximum number of bytes to log for a query result. Defaults to `0` (no limit).

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

### GORM Integration

To run the GORM integration example:
```bash
cd examples/gorm
go run main.go
```
This example demonstrates how to inject the `smartlog` logger into GORM to automatically log SQL queries in the same structured JSON format, including the `log_id` from the request context.

To log the results of queries, you must also register the included GORM plugin:
```go
resultLoggerPlugin := smartlog.NewGormResultLogPlugin(logger, cfg.Gorm)
if err := db.Use(resultLoggerPlugin); err != nil {
    log.Fatalf("Failed to register GORM result logger plugin: %v", err)
}
```
