package smartlog

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// loggingRoundTripper is an http.RoundTripper that logs requests and responses.
type loggingRoundTripper struct {
	next       http.RoundTripper
	logger     *zap.Logger
	redactKeys []string
}

// NewClientLogger creates a new loggingRoundTripper.
func NewClientLogger(next http.RoundTripper, logger *zap.Logger, redactKeys []string) http.RoundTripper {
	return &loggingRoundTripper{
		next:       next,
		logger:     logger,
		redactKeys: redactKeys,
	}
}

// RoundTrip executes a single HTTP transaction, adding logging around it.
func (lrt *loggingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	startTime := time.Now()

	// Get Log ID from context and add to header
	logID, _ := r.Context().Value(LogIDKey).(string)

	ctxLogger := lrt.logger
	if logID != "" {
		r.Header.Set(HeaderLogID, logID)
		ctxLogger = lrt.logger.With(zap.String("log_id", logID))
	}

	// Read and log request body
	var reqBodyBytes []byte
	if r.Body != nil {
		reqBodyBytes, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes)) // Restore body
	}
	redactedReqBody := redactJSONBody(reqBodyBytes, lrt.redactKeys)
	var reqBodyForLog json.RawMessage
	if len(redactedReqBody) > 0 {
		reqBodyForLog = json.RawMessage(redactedReqBody)
	}

	redactedHeaders := redactHeaders(r.Header, lrt.redactKeys)

	ctxLogger.Info("Client request sent",
		zap.String("method", r.Method),
		zap.String("url", r.URL.String()),
		zap.Any("request", map[string]interface{}{
			"headers": redactedHeaders,
			"body":    reqBodyForLog,
		}),
	)

	// Perform the request
	resp, err := lrt.next.RoundTrip(r)
	latency := time.Since(startTime)

	// If there was an error, log it and return
	if err != nil {
		ctxLogger.Error("Client request failed",
			zap.Error(err),
			zap.Int64("latency_ms", latency.Milliseconds()),
		)
		return nil, err
	}

	// Read and log response body
	var respBodyBytes []byte
	if resp.Body != nil {
		respBodyBytes, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes)) // Restore body
	}
	redactedRespBody := redactJSONBody(respBodyBytes, lrt.redactKeys)
	var respBodyForLog json.RawMessage
	if len(redactedRespBody) > 0 {
		respBodyForLog = json.RawMessage(redactedRespBody)
	}

	ctxLogger.Info("Client response received",
		zap.String("method", r.Method),
		zap.String("url", r.URL.String()),
		zap.Int("status", resp.StatusCode),
		zap.Int64("latency_ms", latency.Milliseconds()),
		zap.Any("response", map[string]interface{}{"body": respBodyForLog}),
	)

	return resp, nil
}
