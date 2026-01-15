package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
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

	// --- 3. Echo Instance ---
	e := echo.New()

	// --- 4. Middleware Integration ---
	// Wrap the smartlog middleware for Echo.
	e.Use(echo.WrapMiddleware(smartlog.ServerLogging(logger, cfg.RedactKeys)))

	// --- 5. Example Route ---
	e.POST("/users", func(c echo.Context) error {
		// The logger is available in the request context if needed.
		// Example of how to get it:
		// ctxLogger, _ := c.Request().Context().Value(smartlog.LoggerKey).(*zap.Logger)
		// ctxLogger.Info("Processing user in Echo handler")

		var user struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.Bind(&user); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusCreated, map[string]string{"message": "user created", "username": user.Username})
	})

	fmt.Println("Starting Echo server on :8084")
	fmt.Println("Try sending a request, e.g.:")
	fmt.Println(`curl -X POST -H "Authorization: Bearer secret-token" -d '{"username":"echo-user", "password":"789"}' http://localhost:8084/users`)
	fmt.Println("Then check the console output and 'echo_app.log' file.")

	// --- 6. Start Server ---
	if err := e.Start(":8084"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start Echo server: %v\n", err)
		os.Exit(1)
	}
}
