package smartlog

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormLogger is a custom logger for GORM that integrates with Zap.
type GormLogger struct {
	ZapLogger            *zap.Logger
	LogLevel             logger.LogLevel
	SlowQueryThresholdMs time.Duration
}

// NewGormLogger creates a new GormLogger.
func NewGormLogger(zapLogger *zap.Logger, cfg GormConfig) *GormLogger {
	logLevel := logger.Info
	switch cfg.Level {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	}

	slowQueryThreshold := 200 * time.Millisecond
	if cfg.SlowQueryThresholdMs > 0 {
		slowQueryThreshold = time.Duration(cfg.SlowQueryThresholdMs) * time.Millisecond
	}

	return &GormLogger{
		ZapLogger:            zapLogger,
		LogLevel:             logLevel,
		SlowQueryThresholdMs: slowQueryThreshold,
	}
}

// LogMode sets the log level.
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs informational messages.
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.getLogger(ctx).Info(msg, zap.Any("data", data))
	}
}

// Warn logs warning messages.
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.getLogger(ctx).Warn(msg, zap.Any("data", data))
	}
}

// Error logs error messages.
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.getLogger(ctx).Error(msg, zap.Any("data", data))
	}
}

// Trace logs SQL queries.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []zap.Field{
		zap.Duration("latency", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	logger := l.getLogger(ctx)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("GORM Trace", append(fields, zap.Error(err))...)
	} else if elapsed > l.SlowQueryThresholdMs {
		logger.Warn("GORM Trace (Slow Query)", fields...)
	} else {
		logger.Info("GORM Trace", fields...)
	}
}

// getLogger retrieves the logger from the context or returns the base logger.
func (l *GormLogger) getLogger(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(LoggerKey).(*zap.Logger); ok {
			return logger
		}
	}
	return l.ZapLogger
}
