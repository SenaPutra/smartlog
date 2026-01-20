package smartlog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// LoggerKey is the key for the logger in the request context.
	LoggerKey contextKey = "logger"
	// LogIDKey is the key for the log ID in the request context.
	LogIDKey contextKey = "log_id"
	// HeaderLogID is the name of the header for the log ID.
	HeaderLogID = "X-Request-ID"
)

// responseWriter is a wrapper around http.ResponseWriter to capture the status code and response body.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           new(bytes.Buffer),
	}
}

// WriteHeader captures the status code before writing it to the original ResponseWriter.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response body before writing it to the original ResponseWriter.
func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// ServerLogging is a middleware that logs incoming HTTP requests and their responses.
func ServerLogging(logger *zap.Logger, cfg *Config) func(http.Handler) http.Handler {
	// Create a map for quick lookup of skip paths
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If the path is in our skip list, just call the next handler
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			startTime := time.Now()

			// Get or create Log ID
			logID := r.Header.Get(HeaderLogID)
			if logID == "" {
				logID = uuid.NewString()
			}

			// Create a logger with the log ID
			ctxLogger := logger.With(zap.String("log_id", logID))

			// Add logger and logID to context
			ctx := context.WithValue(r.Context(), LoggerKey, ctxLogger)
			ctx = context.WithValue(ctx, LogIDKey, logID)
			r = r.WithContext(ctx)

			// Read request body
			var reqBodyBytes []byte
			if r.Body != nil {
				reqBodyBytes, _ = io.ReadAll(r.Body)
				// Restore the body so the next handler can read it
				r.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
			}

			// Redact and prepare request body for logging
			redactedReqBody := redactJSONBody(reqBodyBytes, cfg.RedactKeys)
			var reqBodyForLog json.RawMessage
			if len(redactedReqBody) > 0 {
				reqBodyForLog = json.RawMessage(redactedReqBody)
			}

			redactedHeaders := redactHeaders(r.Header, cfg.RedactKeys)

			ctxLogger.Info("Request received",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Any("request", map[string]interface{}{
					"headers": redactedHeaders,
					"body":    reqBodyForLog,
				}),
			)

			// Wrap response writer to capture status and body
			rw := newResponseWriter(w)

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate latency
			latency := time.Since(startTime)

			// Redact and prepare response body for logging
			redactedRespBody := redactJSONBody(rw.body.Bytes(), cfg.RedactKeys)
			var respBodyForLog json.RawMessage
			if len(redactedRespBody) > 0 {
				respBodyForLog = json.RawMessage(redactedRespBody)
			}

			ctxLogger.Info("Response sent",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rw.statusCode),
				zap.Int64("latency_ms", latency.Milliseconds()),
				zap.Any("response", map[string]interface{}{"body": respBodyForLog}),
				zap.Error(nil), // Placeholder for actual error logging
			)
		})
	}
}
