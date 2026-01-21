package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"smartlog"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	// --- 1. Load Configuration ---
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".") // Look for config in the current directory

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	var cfg smartlog.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	// --- 2. Logger Initialization ---
	logger := smartlog.NewLogger(&cfg)
	defer logger.Sync() // Flushes buffer, important for ensuring logs are written

	// --- 3. Mock External Service (Downstream) ---
	// This service will be called by our main service to demonstrate client logging.
	go func() {
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the X-Request-ID header was passed along.
			logID := r.Header.Get(smartlog.HeaderLogID)
			fmt.Printf("[Mock Service] Received request with %s: %s\n", smartlog.HeaderLogID, logID)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"hello from downstream"}`))
		})
		log.Println("[Mock Service] Starting on :8081")
		if err := http.ListenAndServe(":8081", mockHandler); err != nil {
			log.Fatalf("Failed to start mock service: %v", err)
		}
	}()
	// Give the mock service a moment to start
	time.Sleep(100 * time.Millisecond)

	// --- 4. HTTP Client with Logging Middleware ---
	client := &http.Client{
		Transport: smartlog.NewClientLogger(http.DefaultTransport, logger, &cfg),
		Timeout:   5 * time.Second,
	}

	// --- 5. Main Service Handler ---
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The logger and log_id are retrieved from the context,
		// put there by the ServerLogging middleware.
		ctxLogger, _ := r.Context().Value(smartlog.LoggerKey).(*zap.Logger)
		if ctxLogger != nil {
			// This is how you would log within your business logic
			ctxLogger.Info("Processing user request inside handler")
		}

		// Now, make a request to the downstream service.
		// The client logger will automatically pick up the log_id from the context.
		downstreamReq, err := http.NewRequestWithContext(r.Context(), "GET", "http://localhost:8081/data", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := client.Do(downstreamReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(`{"message":"user created","downstream_response":%s}`, string(body))))
	})

	// --- 6. HTTP Server with Logging Middleware ---
	// Wrap the main handler with the server logging middleware.
	loggedRouter := smartlog.ServerLogging(logger, &cfg)(mainHandler)

	fmt.Println("Starting server on :8080")
	fmt.Println("Try sending a request, e.g.:")
	fmt.Println(`curl -X POST -H "Authorization: Bearer secret-token" -d '{"username":"jules", "password":"123"}' http://localhost:8080/users`)
	fmt.Println("Then check the console output and the 'app.log' file.")

	if err := http.ListenAndServe(":8080", loggedRouter); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
