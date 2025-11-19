package logger

import (
	"ecommerce-be/common/constants"
	"os"

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

// WithFields creates a new logger entry with fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

// WithContext creates a new logger entry with context fields (correlation ID and seller ID)
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
	
	return GetLogger().WithFields(fields)
}

// WithContextAndFields creates a new logger entry with both context and additional fields
func WithContextAndFields(c *gin.Context, fields logrus.Fields) *logrus.Entry {
	if fields == nil {
		fields = logrus.Fields{}
	}
	
	// Add correlation ID if present
	if correlationID, exists := c.Get(constants.CORRELATION_ID_KEY); exists {
		fields["correlationId"] = correlationID
	}
	
	// Add seller ID if present
	if sellerID, exists := c.Get(constants.SELLER_ID_KEY); exists {
		fields["sellerId"] = sellerID
	}
	
	return GetLogger().WithFields(fields)
}

// Debug logs a debug message with fields
func Debug(msg string, fields logrus.Fields) {
	GetLogger().WithFields(fields).Debug(msg)
}

// Info logs an info message with fields
func Info(msg string, fields logrus.Fields) {
	GetLogger().WithFields(fields).Info(msg)
}

// Warn logs a warning message with fields
func Warn(msg string, fields logrus.Fields) {
	GetLogger().WithFields(fields).Warn(msg)
}

// Error logs an error message with fields
func Error(msg string, err error, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	GetLogger().WithFields(fields).Error(msg)
}

// Fatal logs a fatal message with fields and exits
func Fatal(msg string, err error, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	GetLogger().WithFields(fields).Fatal(msg)
}

// DebugWithContext logs a debug message with context and fields
func DebugWithContext(c *gin.Context, msg string, fields logrus.Fields) {
	WithContextAndFields(c, fields).Debug(msg)
}

// InfoWithContext logs an info message with context and fields
func InfoWithContext(c *gin.Context, msg string, fields logrus.Fields) {
	WithContextAndFields(c, fields).Info(msg)
}

// WarnWithContext logs a warning message with context and fields
func WarnWithContext(c *gin.Context, msg string, fields logrus.Fields) {
	WithContextAndFields(c, fields).Warn(msg)
}

// ErrorWithContext logs an error message with context and fields
func ErrorWithContext(c *gin.Context, msg string, err error, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	WithContextAndFields(c, fields).Error(msg)
}
