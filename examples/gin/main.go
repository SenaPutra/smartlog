package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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

	// --- 3. Gin Router ---
	router := gin.New()

	// --- 4. Example Route ---
	router.POST("/users", func(c *gin.Context) {
		// The logger is available in the request context if needed.
		// Example of how to get it:
		// ctxLogger, _ := c.Request.Context().Value(smartlog.LoggerKey).(*zap.Logger)
		// ctxLogger.Info("Processing user in Gin handler")

		var user struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "user created", "username": user.Username})
	})

	fmt.Println("Starting Gin server on :8083")
	fmt.Println("Try sending a request, e.g.:")
	fmt.Println(`curl -X POST -H "Authorization: Bearer secret-token" -d '{"username":"gin-user", "password":"456"}' http://localhost:8083/users`)
	fmt.Println("Then check the console output and 'gin_app.log' file.")

	// --- 5. Middleware Integration & Server Start ---
	// Wrap the Gin router with the smartlog middleware.
	loggedRouter := smartlog.ServerLogging(logger, &cfg)(router)

	// Start the server using the standard http package.
	if err := http.ListenAndServe(":8083", loggedRouter); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
