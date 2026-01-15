package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"smartlog"
)

func main() {
	// --- 1. Load Configuration ---
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

	// --- 2. Logger Initialization ---
	logger := smartlog.NewLogger(&cfg)
	defer logger.Sync()

	// --- 3. Mock Server ---
	go func() {
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logID := r.Header.Get(smartlog.HeaderLogID)
			fmt.Printf("[Mock Server] Received request with %s: %s\n", smartlog.HeaderLogID, logID)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"hello from mock server"}`))
		})
		log.Println("[Mock Server] Starting on :8082")
		if err := http.ListenAndServe(":8082", mockHandler); err != nil {
			log.Fatalf("Failed to start mock server: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	// --- 4. Create an HTTP Client with smartlog Transport ---
	httpClient := &http.Client{
		Transport: smartlog.NewClientLogger(http.DefaultTransport, logger, cfg.RedactKeys),
	}

	// --- 5. Create a Resty Client with the smartlog-enabled HTTP Client ---
	client := resty.New().
		SetTransport(httpClient.Transport).
		SetTimeout(10 * time.Second)

	fmt.Println("Sending request with Resty client...")
	fmt.Println("Check the console output and 'resty_app.log' for logs.")

	// --- 6. Make a Request ---
	// The X-Request-ID will be injected automatically if the context contains it.
	// For a standalone client, you can create a log ID manually.
	resp, err := client.R().
		SetHeader("Authorization", "Bearer secret-resty-token").
		SetBody(`{"user":"resty"}`).
		Post("http://localhost:8082/test")

	if err != nil {
		log.Fatalf("Resty request failed: %v", err)
		os.Exit(1)
	}

	fmt.Printf("\nResponse from server:\nStatus: %s\nBody: %s\n", resp.Status(), resp.String())
}
