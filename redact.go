package smartlog

import (
	"encoding/json"
	"net/http"
	"strings"
)

const redactionPlaceholder = "[REDACTED]"

// redactHeaders creates a copy of http.Header and redacts sensitive keys.
func redactHeaders(headers http.Header, keysToRedact []string) http.Header {
	if len(keysToRedact) == 0 {
		return headers
	}

	redactedHeaders := make(http.Header)
	keyMap := make(map[string]struct{})
	for _, key := range keysToRedact {
		keyMap[strings.ToLower(key)] = struct{}{}
	}

	for key, values := range headers {
		if _, exists := keyMap[strings.ToLower(key)]; exists {
			redactedHeaders[key] = []string{redactionPlaceholder}
		} else {
			redactedHeaders[key] = values
		}
	}
	return redactedHeaders
}

// redact takes a map representing a JSON object and a list of keys to redact.
// It recursively redacts the given keys.
func redact(data map[string]interface{}, keysToRedact []string) map[string]interface{} {
	redactedData := make(map[string]interface{})
	keyMap := make(map[string]struct{})
	for _, key := range keysToRedact {
		keyMap[strings.ToLower(key)] = struct{}{}
	}

	for key, value := range data {
		if _, exists := keyMap[strings.ToLower(key)]; exists {
			redactedData[key] = redactionPlaceholder
			continue
		}

		switch v := value.(type) {
		case map[string]interface{}:
			redactedData[key] = redact(v, keysToRedact)
		case []interface{}:
			var newSlice []interface{}
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					newSlice = append(newSlice, redact(m, keysToRedact))
				} else {
					newSlice = append(newSlice, item)
				}
			}
			redactedData[key] = newSlice
		default:
			redactedData[key] = value
		}
	}
	return redactedData
}

// redactJSONBody takes a JSON body as a byte slice and redacts sensitive keys.
// If the body is not a valid JSON object, it returns the original body.
func redactJSONBody(body []byte, keysToRedact []string) []byte {
	if len(keysToRedact) == 0 || len(body) == 0 {
		return body
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// Not a valid JSON object, return as is.
		return body
	}

	redactedData := redact(data, keysToRedact)

	redactedBody, err := json.Marshal(redactedData)
	if err != nil {
		// Should not happen in practice, but as a fallback, return the original.
		return body
	}

	return redactedBody
}
