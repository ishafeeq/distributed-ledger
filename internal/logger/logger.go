package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global structured logger.
var Log *zap.Logger

// Init initializes the global zap logger with a production configuration.
func Init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	// zap.AddCaller() automatically adds the file and line number to every log.
	logger, err := config.Build(zap.AddCaller())
	if err != nil {
		panic(err)
	}
	
	Log = logger
	// ReplaceGlobals allows using zap.L() or zap.S() anywhere in the project.
	zap.ReplaceGlobals(logger)
}

// Sync flushes any buffered log entries.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
