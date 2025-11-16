package logger

import (
	"os"

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
