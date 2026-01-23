package smartlog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGormWithLogger(t *testing.T, logger *zap.Logger, cfg GormConfig) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: NewGormLogger(logger, cfg),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&TestUser{})
	return db
}

func TestSlowQuery(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	t.Run("Logs slow query when enabled", func(t *testing.T) {
		cfg := GormConfig{SlowQueryThresholdMs: 1, Level: "warn"}
		gormLogger := NewGormLogger(logger, cfg)

		// Simulate a slow query by manually calling Trace
		begin := time.Now().Add(-10 * time.Millisecond)
		gormLogger.Trace(context.Background(), begin, func() (string, int64) {
			return "SELECT 1", 1
		}, nil)

		logFound := false
		for _, log := range recorded.All() {
			if log.Message == "GORM Trace (Slow Query)" {
				logFound = true
				break
			}
		}
		assert.True(t, logFound, "Expected to find a slow query log")
		recorded.TakeAll() // Clear logs for next test
	})

	t.Run("Does not log slow query when under threshold", func(t *testing.T) {
		cfg := GormConfig{SlowQueryThresholdMs: 5000, Level: "info"} // 5 seconds, should be fast enough
		gormLogger := NewGormLogger(logger, cfg)

		gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
			return "SELECT 1", 1
		}, nil)

		for _, log := range recorded.All() {
			assert.NotEqual(t, "GORM Trace (Slow Query)", log.Message)
		}
		recorded.TakeAll()
	})

	t.Run("Uses default when threshold is zero", func(t *testing.T) {
		cfg := GormConfig{SlowQueryThresholdMs: 0}
		gormLogger := NewGormLogger(logger, cfg)
		assert.Equal(t, 200*time.Millisecond, gormLogger.SlowQueryThresholdMs)
	})
}
