package smartlog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestServerLoggingMiddleware(t *testing.T) {
	// Setup a mock logger to capture logs
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	// A simple handler to be wrapped by the middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message":"success","data":{"secret":"value"}}`))
	})

	// Keys to be redacted in this test
	redactKeys := []string{"secret", "Authorization"}

	// Create the middleware
	middleware := ServerLogging(logger, redactKeys)
	wrappedHandler := middleware(testHandler)

	// Create a test request
	reqBody := `{"user":"test","password":"sensitive"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(reqBody))
	req.Header.Set("X-Request-ID", "test-id-123")
	req.Header.Set("Authorization", "Bearer secret-token") // Add sensitive header
	req.Header.Set("Content-Type", "application/json")

	// Record the response
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Assertions for the response
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Assertions for the logs
	if recorded.Len() != 2 {
		t.Fatalf("expected 2 logs (request and response), but got %d", recorded.Len())
	}

	// --- Check Request Log ---
	reqLog := recorded.All()[0]
	if reqLog.Message != "Request received" {
		t.Errorf("unexpected request log message: got '%s'", reqLog.Message)
	}
	fields := reqLog.ContextMap()
	if fields["log_id"] != "test-id-123" {
		t.Errorf("unexpected log_id in request log: got %v", fields["log_id"])
	}

	// Check if headers were redacted and nested correctly
	reqField, ok := fields["request"].(map[string]interface{})
	if !ok {
		t.Fatal("request field is not a map")
	}
	headers, ok := reqField["headers"].(http.Header)
	if !ok {
		t.Fatalf("headers field is not a http.Header, got %T", reqField["headers"])
	}
	if headers.Get("Authorization") != redactionPlaceholder {
		t.Errorf("Authorization header was not redacted: got '%s'", headers.Get("Authorization"))
	}
	if headers.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type header was incorrect: got '%s'", headers.Get("Content-Type"))
	}

	// --- Check Response Log ---
	respLog := recorded.All()[1]
	if respLog.Message != "Response sent" {
		t.Errorf("unexpected response log message: got '%s'", respLog.Message)
	}
	fields = respLog.ContextMap()
	if fields["status"].(int64) != http.StatusCreated {
		t.Errorf("unexpected status in response log: got %v", fields["status"])
	}

	// Check if the response body was redacted correctly
	respBodyField, ok := fields["response"].(map[string]interface{})
	if !ok {
		t.Fatal("response field is not a map")
	}
	respBody, _ := respBodyField["body"].(json.RawMessage)
	var respData map[string]interface{}
	json.Unmarshal(respBody, &respData)
	dataField, _ := respData["data"].(map[string]interface{})

	if dataField["secret"] != redactionPlaceholder {
		t.Errorf("response body was not redacted correctly, got: %s", dataField["secret"])
	}
}

func TestServerLogging_FileCreationAndContent(t *testing.T) {
	// 1. Setup a temporary directory for the log file
	//tempDir := t.TempDir()
	logPath := filepath.Join("test.log")

	// 2. Configure and create a logger that writes to the temp file
	cfg := &Config{
		ServiceName: "test-service",
		Env:         "test",
		Log: TimberjackConfig{
			Filename: logPath,
		},
		RedactKeys: []string{"password"},
	}
	logger := NewLogger(cfg)
	defer logger.Sync()

	// 3. Setup the test handler and middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	middleware := ServerLogging(logger, cfg.RedactKeys)
	wrappedHandler := middleware(testHandler)

	// 4. Send a request to trigger the logger
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// 5. Verify the log file was created
	_, err := os.Stat(logPath)
	require.NoError(t, err, "Log file should be created")

	// 6. Read the content of the log file
	logContent, err := os.ReadFile(logPath)
	require.NoError(t, err, "Should be able to read the log file")

	// 7. Verify the log content
	logString := string(logContent)
	assert.Contains(t, logString, `"message":"Request received"`, "Log should contain the request received message")
	assert.Contains(t, logString, `"message":"Response sent"`, "Log should contain the response sent message")
	assert.Contains(t, logString, `"service":"test-service"`, "Log should contain the service name")
	assert.Contains(t, logString, `"path":"/health"`, "Log should contain the request path")
}
