package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"smartlog"
	"time"

	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User model for GORM
type User struct {
	gorm.Model
	Name  string `json:"name"`
	Email string `json:"email"`
}

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

	// --- 3. GORM Initialization ---
	gormLogger := smartlog.NewGormLogger(logger, cfg.Gorm)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	resultLoggerPlugin := smartlog.NewGormResultLogPlugin(logger, cfg.Gorm)
	if err := db.Use(resultLoggerPlugin); err != nil {
		log.Fatalf("Failed to register GORM result logger plugin: %v", err)
	}
	db.AutoMigrate(&User{})

	// --- 4. HTTP Client with Logging Middleware ---
	client := &http.Client{
		Transport: smartlog.NewClientLogger(http.DefaultTransport, logger, &cfg),
		Timeout:   5 * time.Second,
	}

	// --- 5. Mock Downstream Service ---
	go func() {
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logID := r.Header.Get(smartlog.HeaderLogID)
			fmt.Printf("[Mock Service] Received notification request with %s: %s\n", smartlog.HeaderLogID, logID)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"notification sent"}`))
		})
		log.Println("[Mock Service] Starting on :8086")
		http.ListenAndServe(":8086", mockHandler)
	}()
	time.Sleep(100 * time.Millisecond)

	// --- 6. Main HTTP Server ---
	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxDB := db.WithContext(r.Context())

		// Create a user in the database
		user := User{Name: "Jules", Email: "jules@example.com"}
		if err := ctxDB.Create(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Call the downstream notification service
		downstreamReq, _ := http.NewRequestWithContext(r.Context(), "POST", "http://localhost:8086/notify", nil)
		downstreamReq.Header.Set("Authorization", "Bearer secret-downstream-token")
		client.Do(downstreamReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"message":"user created successfully", "user_id":%d}`, user.ID)
	})

	mux := http.NewServeMux()
	mux.Handle("/users", userHandler)

	loggedRouter := smartlog.ServerLogging(logger, &cfg)(mux)

	fmt.Println("Starting end-to-end example server on :8085")
	fmt.Println("Try sending a request to see the full log trace:")
	fmt.Println(`curl -X POST http://localhost:8085/users`)
	if err := http.ListenAndServe(":8085", loggedRouter); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
