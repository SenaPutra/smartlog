package smartlog

import (
	"bytes"
	"testing"
)

func TestRedactJSONBody(t *testing.T) {
	testCases := []struct {
		name         string
		inputBody    []byte
		keysToRedact []string
		expectedBody []byte
	}{
		{
			name:         "No redaction keys",
			inputBody:    []byte(`{"user":"jules","password":"supersecret"}`),
			keysToRedact: []string{},
			expectedBody: []byte(`{"user":"jules","password":"supersecret"}`),
		},
		{
			name:         "Simple redaction",
			inputBody:    []byte(`{"user":"jules","password":"supersecret"}`),
			keysToRedact: []string{"password"},
			expectedBody: []byte(`{"password":"[REDACTED]","user":"jules"}`),
		},
		{
			name:         "Case-insensitive redaction",
			inputBody:    []byte(`{"user":"jules","Authorization":"Bearer token"}`),
			keysToRedact: []string{"authorization"},
			expectedBody: []byte(`{"Authorization":"[REDACTED]","user":"jules"}`),
		},
		{
			name:         "Nested redaction",
			inputBody:    []byte(`{"user":{"name":"jules","details":{"token":"secret-token"}}}`),
			keysToRedact: []string{"token"},
			expectedBody: []byte(`{"user":{"details":{"token":"[REDACTED]"},"name":"jules"}}`),
		},
		{
			name:         "Redaction in an array of objects",
			inputBody:    []byte(`{"users":[{"name":"jules","api_key":"key1"},{"name":"agent","api_key":"key2"}]}`),
			keysToRedact: []string{"api_key"},
			expectedBody: []byte(`{"users":[{"api_key":"[REDACTED]","name":"jules"},{"api_key":"[REDACTED]","name":"agent"}]}`),
		},
		{
			name:         "Invalid JSON input",
			inputBody:    []byte(`not a json`),
			keysToRedact: []string{"password"},
			expectedBody: []byte(`not a json`),
		},
		{
			name:         "Empty JSON body",
			inputBody:    []byte(`{}`),
			keysToRedact: []string{"password"},
			expectedBody: []byte(`{}`),
		},
		{
			name:         "Empty input body",
			inputBody:    []byte(``),
			keysToRedact: []string{"password"},
			expectedBody: []byte(``),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := redactJSONBody(tc.inputBody, tc.keysToRedact)
			if !bytes.Equal(result, tc.expectedBody) {
				t.Errorf("Expected '%s', but got '%s'", tc.expectedBody, result)
			}
		})
	}
}
