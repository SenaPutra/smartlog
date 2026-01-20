package smartlog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestUser struct {
	gorm.Model
	Name string
}

func setupGormWithPlugin(t *testing.T, logger *zap.Logger, cfg GormConfig) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: NewGormLogger(logger, cfg),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	plugin := NewGormResultLogPlugin(logger, cfg)
	if err := db.Use(plugin); err != nil {
		t.Fatalf("Failed to register GORM plugin: %v", err)
	}

	db.AutoMigrate(&TestUser{})
	return db
}

func TestGormResultLogPlugin(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	t.Run("Logs result when enabled", func(t *testing.T) {
		cfg := GormConfig{LogQueryResult: true, LogResultMaxBytes: 1024}
		db := setupGormWithPlugin(t, logger, cfg)

		user := TestUser{Name: "jules-test"}
		db.Create(&user)
		var foundUser TestUser
		db.First(&foundUser, user.ID)

		// Check the logs
		logFound := false
		for _, log := range recorded.All() {
			if log.Message == "GORM Query Result" {
				logFound = true
				resultField, ok := log.ContextMap()["result"].(string)
				assert.True(t, ok, "Result field should be a string")
				assert.Contains(t, resultField, `"Name":"jules-test"`)
			}
		}
		assert.True(t, logFound, "Expected to find GORM Query Result log")
		recorded.TakeAll() // Clear logs for next test
	})

	t.Run("Does not log result when disabled", func(t *testing.T) {
		cfg := GormConfig{LogQueryResult: false}
		db := setupGormWithPlugin(t, logger, cfg)

		user := TestUser{Name: "jules-disabled"}
		db.Create(&user)
		var foundUser TestUser
		db.First(&foundUser, user.ID)

		// Check the logs
		for _, log := range recorded.All() {
			assert.NotEqual(t, "GORM Query Result", log.Message)
		}
		recorded.TakeAll()
	})

	t.Run("Truncates result when it exceeds max bytes", func(t *testing.T) {
		// Set a limit that will include ID and CreatedAt, but not Name
		cfg := GormConfig{LogQueryResult: true, LogResultMaxBytes: 70}
		db := setupGormWithPlugin(t, logger, cfg)

		user := TestUser{Name: "a-very-long-name-to-test-truncation"}
		db.Create(&user)
		var foundUser TestUser
		db.First(&foundUser, user.ID)

		// Check the logs
		logFound := false
		for _, log := range recorded.All() {
			if log.Message == "GORM Query Result" {
				logFound = true
				resultField, ok := log.ContextMap()["result"].(string)
				assert.True(t, ok, "Result field should be a string")

				// Check that it's valid JSON
				assert.True(t, json.Valid([]byte(resultField)), "Truncated result should be valid JSON")

				// Check that it contains some of the initial fields but not all
				assert.Contains(t, resultField, `"ID":`, "Should contain the ID field")
				assert.Contains(t, resultField, `"CreatedAt":`, "Should contain the CreatedAt field")
				assert.NotContains(t, resultField, `"Name":`, "Should not contain the Name field due to truncation")
			}
		}
		assert.True(t, logFound, "Expected to find GORM Query Result log")
		recorded.TakeAll()
	})

	t.Run("Handles context correctly", func(t *testing.T) {
		cfg := GormConfig{LogQueryResult: true}
		db := setupGormWithPlugin(t, logger, cfg)

		// Create a logger with a log_id
		ctxLogger := logger.With(zap.String("log_id", "test-gorm-log-id"))
		ctx := context.WithValue(context.Background(), LoggerKey, ctxLogger)
		dbWithCtx := db.WithContext(ctx)

		user := TestUser{Name: "context-user"}
		dbWithCtx.Create(&user)
		var foundUser TestUser
		dbWithCtx.First(&foundUser, user.ID)

		// Check that the log_id is in the GORM result log
		logFound := false
		for _, log := range recorded.All() {
			if log.Message == "GORM Query Result" {
				logFound = true
				logID, ok := log.ContextMap()["log_id"].(string)
				assert.True(t, ok)
				assert.Equal(t, "test-gorm-log-id", logID)
			}
		}
		assert.True(t, logFound, "Expected to find GORM Query Result log with log_id")
		recorded.TakeAll()
	})
}
