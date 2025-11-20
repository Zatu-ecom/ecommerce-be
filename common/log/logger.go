package log

import (
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
func WithContext(c *gin.Context) *logrus.Entry {
	fields := logrus.Fields{}

	// Add correlation ID if present
	if correlationID, exists := c.Get(constants.CORRELATION_ID_KEY); exists {
		fields["correlationId"] = correlationID
	}

	// Add seller ID if present
	if sellerID, exists := c.Get(constants.SELLER_ID_KEY); exists {
		fields["sellerId"] = sellerID
	}

	// Add user ID if present
	if userID, exists := c.Get(constants.USER_ID_KEY); exists {
		fields["userId"] = userID
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

// DebugWithContext logs a debug message with context
func DebugWithContext(c *gin.Context, msg string) {
	WithContext(c).Debug(msg)
}

// InfoWithContext logs an info message with context
func InfoWithContext(c *gin.Context, msg string) {
	WithContext(c).Info(msg)
}

// WarnWithContext logs a warning message with context
func WarnWithContext(c *gin.Context, msg string) {
	WithContext(c).Warn(msg)
}

// ErrorWithContext logs an error message with context and error details
func ErrorWithContext(c *gin.Context, msg string, err error) {
	entry := WithContext(c)
	if err != nil {
		entry.WithField("error", err.Error()).Error(msg)
	} else {
		entry.Error(msg)
	}
}
