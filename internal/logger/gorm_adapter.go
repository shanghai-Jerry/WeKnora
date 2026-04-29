package logger

import (
	"context"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

// GormLoggerAdapter adapts the internal logger to GORM's logger.Interface.
type GormLoggerAdapter struct {
	skipErrRecordNotFound bool
	slowThreshold         time.Duration
}

// NewGormLogger creates a GORM logger adapter using the internal logger.
// If slowThreshold is 0, slow SQL logging is disabled.
func NewGormLogger(slowThreshold time.Duration) gormlogger.Interface {
	return &GormLoggerAdapter{
		skipErrRecordNotFound: true,
		slowThreshold:         slowThreshold,
	}
}

func (a *GormLoggerAdapter) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	// The internal logger controls log level via LOG_LEVEL env var.
	// Return a copy to satisfy the interface.
	return &GormLoggerAdapter{
		skipErrRecordNotFound: a.skipErrRecordNotFound,
		slowThreshold:         a.slowThreshold,
	}
}

func (a *GormLoggerAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	Infof(ctx, msg, data...)
}

func (a *GormLoggerAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	Warnf(ctx, msg, data...)
}

func (a *GormLoggerAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	Errorf(ctx, msg, data...)
}

func (a *GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)

	// Skip ErrRecordNotFound if configured
	if err != nil && a.skipErrRecordNotFound {
		if err == gormlogger.ErrRecordNotFound {
			return
		}
		Errorf(ctx, "[GORM] %v (elapsed: %s)", err, elapsed)
		return
	}

	sql, rows := fc()

	// Log slow queries if threshold is set
	if a.slowThreshold > 0 && elapsed > a.slowThreshold {
		Warnf(ctx, "[GORM] SLOW SQL >= %s | %s | %s | rows: %d",
			a.slowThreshold, elapsed, sql, rows)
		return
	}

	// Log all SQL at debug level (only if debug enabled)
	Debugf(ctx, "[GORM] %s | %s | rows: %d", elapsed, sql, rows)
}
