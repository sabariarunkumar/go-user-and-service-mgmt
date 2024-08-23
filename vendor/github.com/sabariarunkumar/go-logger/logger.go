package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// timestampFormat RFC Time
	timestampFormat = "15:04:05.999 02/01/2006 (MST)"
	// configuredLogLevelError error
	configuredLogLevelError = "error"
	// configuredLogLevelInfo info
	configuredLogLevelInfo = "info"
	// configuredLogLevelDebug debug
	configuredLogLevelDebug = "debug"
)

// customTimeEncoder...
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(timestampFormat))
}

// getZapLogger configures zap logger
func getZapLogger(level zapcore.Level) *zap.Logger {
	// encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = customTimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	zapEncoder := zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), level)
	return zap.New(zapEncoder, zap.AddCallerSkip(0), zap.AddCaller())
}

// NewLogger initializes zap logger with log level
func NewLogger(logLevel string) *zap.SugaredLogger {
	var zapLevel zapcore.Level
	switch logLevel {
	case configuredLogLevelError:
		zapLevel = zap.ErrorLevel
	case configuredLogLevelDebug:
		zapLevel = zap.DebugLevel
	case configuredLogLevelInfo:
		zapLevel = zap.InfoLevel
	default:
		// let us consider info level by default
		zapLevel = zap.InfoLevel
	}
	return getZapLogger(zapLevel).Sugar()
}
