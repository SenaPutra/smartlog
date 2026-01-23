package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"smartlog"

	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User model for GORM
type User struct {
	gorm.Model
	Name string
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

	// Register the result logging plugin
	resultLoggerPlugin := smartlog.NewGormResultLogPlugin(logger, cfg.Gorm)
	if err := db.Use(resultLoggerPlugin); err != nil {
		log.Fatalf("Failed to register GORM result logger plugin: %v", err)
	}

	db.AutoMigrate(&User{})

	// --- 4. HTTP Server ---
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pass the request context to GORM
		ctxDB := db.WithContext(r.Context())

		// Create a user
		user := User{Name: "Jules"}
		if err := ctxDB.Create(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Find the user
		var foundUser User
		if err := ctxDB.First(&foundUser, user.ID).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "User created and found: %s", foundUser.Name)
	})

	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	mux := http.NewServeMux()
	mux.Handle("/users", handler)
	mux.Handle("/health", healthHandler)

	loggedRouter := smartlog.ServerLogging(logger, &cfg)(mux)

	fmt.Println("Starting server on :8085")
	fmt.Println("Try sending a request to http://localhost:8085/users to see GORM logs.")
	fmt.Println("Try sending a request to http://localhost:8085/health to see skipped logs.")
	if err := http.ListenAndServe(":8085", loggedRouter); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
