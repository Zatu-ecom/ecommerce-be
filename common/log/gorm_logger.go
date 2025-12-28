package log

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger is a custom GORM logger that outputs JSON format using logrus
type GormLogger struct {
	LogLevel                  gormlogger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

// NewGormLogger creates a new GormLogger instance with log level based on application config
func NewGormLogger() *GormLogger {
	return &GormLogger{
		LogLevel:                  getGormLogLevel(),
		SlowThreshold:             200 * time.Millisecond, // Queries slower than this are logged as warn
		IgnoreRecordNotFoundError: true,
	}
}

// getGormLogLevel converts application log level to GORM log level
func getGormLogLevel() gormlogger.LogLevel {
	if Log == nil {
		return gormlogger.Info
	}

	switch Log.GetLevel() {
	case logrus.DebugLevel:
		return gormlogger.Info // GORM Info shows all SQL queries
	case logrus.InfoLevel:
		return gormlogger.Info
	case logrus.WarnLevel:
		return gormlogger.Warn
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return gormlogger.Error
	default:
		return gormlogger.Info
	}
}

// LogMode sets the log level
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs info level messages
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		WithContext(ctx).WithFields(logrus.Fields{
			"component": "gorm",
		}).Infof(msg, data...)
	}
}

// Warn logs warn level messages
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		WithContext(ctx).WithFields(logrus.Fields{
			"component": "gorm",
		}).Warnf(msg, data...)
	}
}

// Error logs error level messages
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		WithContext(ctx).WithFields(logrus.Fields{
			"component": "gorm",
		}).Errorf(msg, data...)
	}
}

// Trace logs SQL queries with execution details
func (l *GormLogger) Trace(
	ctx context.Context,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
	err error,
) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Build base fields
	fields := logrus.Fields{
		"component":     "gorm",
		"duration_ms":   float64(elapsed.Nanoseconds()) / 1e6,
		"rows_affected": rows,
		"sql":           sql,
	}

	// Determine log level based on error and duration
	switch {
	case err != nil && l.LogLevel >= gormlogger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		fields["error"] = err.Error()
		WithContext(ctx).WithFields(fields).Error("SQL query failed")

	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		fields["slow_query"] = true
		fields["threshold_ms"] = float64(l.SlowThreshold.Nanoseconds()) / 1e6
		WithContext(ctx).WithFields(fields).Warn("Slow SQL query detected")

	case l.LogLevel == gormlogger.Info:
		WithContext(ctx).WithFields(fields).Debug("SQL query executed")
	}
}

// ParamsFilter returns the parameters filter function (not used but required by interface)
func (l *GormLogger) ParamsFilter(
	ctx context.Context,
	sql string,
	params ...interface{},
) (string, []interface{}) {
	return sql, params
}

// FormatSQL formats SQL for logging (helper function)
func FormatSQL(sql string, rows int64, elapsed time.Duration) string {
	return fmt.Sprintf("[%.3fms] [rows:%d] %s", float64(elapsed.Nanoseconds())/1e6, rows, sql)
}
