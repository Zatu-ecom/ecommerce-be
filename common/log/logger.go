package log

import (
	"context"
	"os"

	"ecommerce-be/common/constants"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// InitLogger initializes the global logger instance
func InitLogger() {
	Log = logrus.New()

	// Set JSON formatter for structured logging
	Log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
		PrettyPrint: true,
	})

	// Set output to stdout
	Log.SetOutput(os.Stdout)

	// Set log level from environment variable or default to Info
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		Log.SetLevel(logrus.DebugLevel)
	case "warn":
		Log.SetLevel(logrus.WarnLevel)
	case "error":
		Log.SetLevel(logrus.ErrorLevel)
	default:
		Log.SetLevel(logrus.InfoLevel)
	}

	Log.Info("Logger initialized")
}

// GetLogger returns the global logger instance
func GetLogger() *logrus.Logger {
	if Log == nil {
		InitLogger()
	}
	return Log
}

// WithContext creates a new logger entry with context fields (correlation ID, seller ID, user ID)
// Works with both standard context.Context and *gin.Context (which embeds context.Context)
func WithContext(ctx context.Context) *logrus.Entry {
	fields := logrus.Fields{}

	// Try to get as Gin context first (for values stored via c.Set())
	if ginCtx, ok := ctx.(*gin.Context); ok {
		// Extract from Gin context storage
		if correlationID, exists := ginCtx.Get(constants.CORRELATION_ID_KEY); exists {
			fields["correlationId"] = correlationID
		}
		if sellerID, exists := ginCtx.Get(constants.SELLER_ID_KEY); exists {
			fields["sellerId"] = sellerID
		}
		if userID, exists := ginCtx.Get(constants.USER_ID_KEY); exists {
			fields["userId"] = userID
		}
	} else {
		// Extract from standard context.Context (for background jobs, tests, etc.)
		if correlationID := ctx.Value(constants.CORRELATION_ID_KEY); correlationID != nil {
			fields["correlationId"] = correlationID
		}
		if sellerID := ctx.Value(constants.SELLER_ID_KEY); sellerID != nil {
			fields["sellerId"] = sellerID
		}
		if userID := ctx.Value(constants.USER_ID_KEY); userID != nil {
			fields["userId"] = userID
		}
	}

	return GetLogger().WithFields(fields)
}

// Debug logs a debug message
func Debug(msg string) {
	GetLogger().Debug(msg)
}

// Info logs an info message
func Info(msg string) {
	GetLogger().Info(msg)
}

// Warn logs a warning message
func Warn(msg string) {
	GetLogger().Warn(msg)
}

// Error logs an error message with error details
func Error(msg string, err error) {
	if err != nil {
		GetLogger().WithField("error", err.Error()).Error(msg)
	} else {
		GetLogger().Error(msg)
	}
}

// Fatal logs a fatal message with error details and exits
func Fatal(msg string, err error) {
	if err != nil {
		GetLogger().WithField("error", err.Error()).Fatal(msg)
	} else {
		GetLogger().Fatal(msg)
	}
}

// DebugWithContext logs a debug message with context (generic version)
func DebugWithContext(ctx context.Context, msg string) {
	WithContext(ctx).Debug(msg)
}

// InfoWithContext logs an info message with context (generic version)
func InfoWithContext(ctx context.Context, msg string) {
	WithContext(ctx).Info(msg)
}

// WarnWithContext logs a warning message with context (generic version)
func WarnWithContext(ctx context.Context, msg string) {
	WithContext(ctx).Warn(msg)
}

// ErrorWithContext logs an error message with context and error details (generic version)
func ErrorWithContext(ctx context.Context, msg string, err error) {
	entry := WithContext(ctx)
	if err != nil {
		entry.WithField("error", err.Error()).Error(msg)
	} else {
		entry.Error(msg)
	}
}
