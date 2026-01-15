package smartlog

import (
	"os"
	"time"

	"github.com/DeRuina/timberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new Zap logger with Timberjack for log rotation.
func NewLogger(cfg *Config) *zap.Logger {
	// Timberjack hook for rotating log files
	timberjackHook := &timberjack.Logger{
		Filename:         cfg.Log.Filename,
		MaxSize:          cfg.Log.MaxSize,
		MaxBackups:       cfg.Log.MaxBackups,
		MaxAge:           cfg.Log.MaxAge,
		Compression:      cfg.Log.Compression,
		RotationInterval: time.Duration(cfg.Log.RotationInterval) * time.Hour,
	}

	// Zap core configuration
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.MessageKey = "message"

	// Create a core that writes to the timberjack hook
	fileWriter := zapcore.AddSync(timberjackHook)
	// Also create a core that writes to the console
	consoleWriter := zapcore.AddSync(os.Stdout)

	// Combine writers to log to both file and console
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), fileWriter, zap.InfoLevel),
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), consoleWriter, zap.DebugLevel),
	)

	// Create the logger with the service and env fields
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).
		With(
			zap.String("service", cfg.ServiceName),
			zap.String("env", cfg.Env),
		)

	return logger
}
