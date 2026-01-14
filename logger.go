package smartlog

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger creates a new Zap logger with Lumberjack for log rotation.
func NewLogger(cfg *Config) *zap.Logger {
	// Lumberjack hook for rotating log files
	lumberjackHook := &lumberjack.Logger{
		Filename:   cfg.LogPath,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     7, //days
		Compress:   true,
	}

	// Zap core configuration
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.MessageKey = "message"

	// Create a core that writes to the lumberjack hook
	fileWriter := zapcore.AddSync(lumberjackHook)
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
