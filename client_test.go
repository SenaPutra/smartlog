package smartlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// mockRoundTripper is a mock implementation of http.RoundTripper for testing.
type mockRoundTripper struct {
	roundTripFunc func(r *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return m.roundTripFunc(r)
}

func TestClientLoggingMiddleware(t *testing.T) {
	// Setup a mock logger to capture logs
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	// Keys to be redacted in this test
	cfg := &Config{
		RedactKeys: []string{"Api-Key"},
	}

	// Mock the downstream server's response
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(r *http.Request) (*http.Response, error) {
			// Check if the client middleware passed the headers correctly
			if r.Header.Get(HeaderLogID) != "client-log-id" {
				t.Errorf("Expected header %s to be 'client-log-id', got '%s'", HeaderLogID, r.Header.Get(HeaderLogID))
			}
			if r.Header.Get("Api-Key") != "secret-api-key" {
				t.Errorf("Expected header Api-Key to be 'secret-api-key', got '%s'", r.Header.Get("Api-Key"))
			}
			return httptest.NewRecorder().Result(), nil
		},
	}

	// Create the client with the logging middleware
	client := &http.Client{
		Transport: NewClientLogger(mockTransport, logger, cfg),
	}

	// Create a request with context containing the log ID
	ctx := context.WithValue(context.Background(), LogIDKey, "client-log-id")
	req, err := http.NewRequestWithContext(ctx, "GET", "http://downstream.example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Api-Key", "secret-api-key")
	req.Header.Set("Accept", "application/json")

	// Perform the request
	_, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	// Assertions for the logs
	if recorded.Len() != 2 {
		t.Fatalf("expected 2 logs (client request and response), but got %d", recorded.Len())
	}

	// --- Check Client Request Log ---
	reqLog := recorded.All()[0]
	if reqLog.Message != "Client request sent" {
		t.Errorf("unexpected log message: got '%s'", reqLog.Message)
	}

	fields := reqLog.ContextMap()
	if fields["log_id"] != "client-log-id" {
		t.Errorf("unexpected log_id in log: got %v", fields["log_id"])
	}

	// Check if headers were redacted
	reqField, ok := fields["request"].(map[string]interface{})
	if !ok {
		t.Fatal("request field is not a map")
	}
	headers, ok := reqField["headers"].(http.Header)
	if !ok {
		t.Fatalf("headers field is not a http.Header, got %T", reqField["headers"])
	}
	if headers.Get("Api-Key") != redactionPlaceholder {
		t.Errorf("Api-Key header was not redacted: got '%s'", headers.Get("Api-Key"))
	}
	if headers.Get("Accept") != "application/json" {
		t.Errorf("Accept header was incorrect: got '%s'", headers.Get("Accept"))
	}
}
