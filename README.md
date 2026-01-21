# smartlog

`smartlog` is a flexible and easy-to-use logging middleware for Go's `net/http` servers and clients. It's built on top of the high-performance [Zap](https://github.com/uber-go/zap) logger and uses [Timberjack](https://github.com/DeRuina/timberjack) for advanced, time-based log rotation.

The main goal is to provide a "plug-and-play" solution for structured JSON logging that can be easily integrated into any Go web application, tracing a single request from server ingress, through database queries, to downstream client requests.

## Features

- **Structured JSON Logging**: All logs are in JSON format, making them easy to parse, search, and analyze.
- **End-to-End Traceability**: Traces the entire lifecycle of a request using a consistent `log_id` (`X-Request-ID`), from the moment it hits your server, through GORM database queries, to any downstream API calls it makes.
- **Request & Response Logging**: Automatically logs details of incoming server requests and outgoing client requests.
- **GORM Integration**: Provides a custom GORM logger to automatically log SQL queries in the same structured JSON format.
- **Configurable Log Rotation**: Uses Timberjack to handle time-based log rotation, compression, and cleanup automatically.
- **Dynamic Log Levels**: Configure different log levels for file and console output.
- **Skippable Routes**: Exclude noisy endpoints (like `/health` or `/metrics`) from logging.
- **Sensitive Data Redaction**: Automatically redacts sensitive data from log bodies and headers.
- **Generic Middleware**: Designed to work with any `http.Handler` and `http.RoundTripper`, making it highly compatible with frameworks like Gin, Echo, Chi, and clients like Resty.

## Installation

```bash
go get github.com/your-username/smartlog
```
*(Note: Replace `your-username` with the actual path once it's published)*

## Configuration

`smartlog` is configured via a single YAML file (e.g., `config.yml`), loaded with [Viper](https://github.com/spf13/viper). This approach keeps your logging setup clean and easy to manage.

**Example `config.yml`:**
```yaml
service_name: "example-service"
env: "development"
redact_keys: ["password", "Authorization", "token"]
skip_paths: ["/health", "/metrics"]

log:
  filename: "app.log"
  max_size: 10
  max_backups: 3
  max_age: 7
  compression: "gzip"
  rotation_interval: 24 # in hours
  level: "debug"        # "debug", "info", "warn", "error"

gorm:
  level: "info"                 # "silent", "error", "warn", "info"
  log_query_result: true        # Log data returned from queries
  log_result_max_bytes: 1024    # Truncate large results
```

### Configuration Details
- `service_name`: The name of your service (e.g., "user-service").
- `env`: The environment (e.g., "production", "development").
- `redact_keys`: A list of keys to be censored in logs.
- `skip_paths`: A list of URL paths to exclude from logging.
- `log`:
  - `filename`: The path for the log file.
  - `max_size`, `max_backups`, `max_age`: Standard log rotation settings.
  - `compression`: Compression for rotated logs ("gzip" or "none").
  - `rotation_interval`: The rotation interval in hours (e.g., 24 for daily).
  - `level`: Log level for the file logger. Defaults to "info".
- `gorm`:
  - `level`: Log level for GORM's logger. Defaults to "info".
  - `log_query_result`: Set to `true` to log data returned from queries. Defaults to `false`.
  - `log_result_max_bytes`: Max bytes for a logged query result.

## Usage

### 1. Initializing the Logger
Load your configuration and create a logger instance once when your application starts.

```go
import (
    "log"
    "smartlog"
    "github.com/spf13/viper"
)

// Load config from file
viper.SetConfigName("config")
viper.SetConfigType("yml")
viper.AddConfigPath(".")
if err := viper.ReadInConfig(); err != nil {
    log.Fatalf("Error reading config file: %s", err)
}

var cfg smartlog.Config
if err := viper.Unmarshal(&cfg); err != nil {
    log.Fatalf("Unable to decode into struct: %v", err)
}

// Create logger
logger := smartlog.NewLogger(&cfg)
defer logger.Sync() // Flushes the buffer
```

### 2. Server Logging Middleware
Wrap your main router or handler with the `ServerLogging` middleware.

```go
// myRouter can be any http.Handler (e.g., http.NewServeMux, Gin, Chi)
loggedRouter := smartlog.ServerLogging(logger, &cfg)(myRouter)
http.ListenAndServe(":8080", loggedRouter)
```

### 3. Client Logging Middleware
Create an `http.Client` and set its `Transport` to the `NewClientLogger`.

```go
client := &http.Client{
    Transport: smartlog.NewClientLogger(http.DefaultTransport, logger, &cfg),
}

// All requests made with this client will now be logged
// and will carry the log_id from the incoming request's context.
req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/data", nil)
resp, err := client.Do(req)
```

### 4. GORM Integration
Inject `smartlog` into GORM to automatically log SQL queries.

```go
import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// Create the GORM logger
gormLogger := smartlog.NewGormLogger(logger, cfg.Gorm)

// Initialize GORM
db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
    Logger: gormLogger,
})

// To log query results, register the plugin
resultLoggerPlugin := smartlog.NewGormResultLogPlugin(logger, cfg.Gorm)
if err := db.Use(resultLoggerPlugin); err != nil {
    log.Fatalf("Failed to register GORM plugin: %v", err)
}

// Now, GORM operations will be logged automatically
// db.WithContext(ctx).First(&user, 1)
```

## Running the Examples

The `examples/` directory contains several runnable examples.

### End-to-End Example (Recommended)
This is the best place to start. It demonstrates the full power of `smartlog`, showing how a single `log_id` traces a request from the server, through a GORM query, to a downstream client call.

```bash
cd examples/end-to-end
go run main.go
```
Then, from another terminal, send a request:
```bash
curl -X POST http://localhost:8085/users
```
Check `e2e_app.log` and filter by a `log_id` to see the complete trace.

### Other Integration Examples
- **Resty Client:** `cd examples/resty && go run main.go`
- **Gin Server:** `cd examples/gin && go run main.go`
- **Echo Server:** `cd examples/echo && go run main.go`
- **GORM (standalone):** `cd examples/gorm && go run main.go`
