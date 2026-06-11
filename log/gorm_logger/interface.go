package gorm_logger

import (
	"context"
	"fmt"
	"time"

	"github.com/lwy110193/go_vendor/log"
	gorm_logger "gorm.io/gorm/logger"
)

// GormLogger GORM 日志接口实现
type GormLogger struct {
	Logger    log.LoggerInterface
	level     gorm_logger.LogLevel
	slowQuery time.Duration
}

// NewGormLogger 创建一个新的 GORM 日志记录器
func NewGormLogger(logger log.LoggerInterface) *GormLogger {
	return &GormLogger{
		Logger: logger,
		level:  gorm_logger.Info, // 默认为 Info 级别
	}
}

// LogMode 设置日志级别并返回新的日志记录器
func (g *GormLogger) LogMode(level gorm_logger.LogLevel) gorm_logger.Interface {
	newLogger := *g
	newLogger.level = level
	return &newLogger
}

// Info 记录信息级别日志
func (g *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if g.level >= gorm_logger.Info {
		g.Logger.InfoCtx(ctx, msg, data...)
	}
}

// Warn 记录警告级别日志
func (g *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if g.level >= gorm_logger.Warn {
		g.Logger.WarnCtx(ctx, msg, data...)
	}
}

// Error 记录错误级别日志
func (g *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if g.level >= gorm_logger.Error {
		g.Logger.ErrorCtx(ctx, msg, data...)
	}
}

// Trace 记录 SQL 执行信息
func (g *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if g.level <= gorm_logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && g.level >= gorm_logger.Error:
		g.Logger.ErrorCtx(ctx, fmt.Sprintf("SQL执行错误( err: %v , sql: %v )", err.Error(), sql))
	case elapsed > g.slowQuery && g.level >= gorm_logger.Warn:
		g.Logger.WarnCtx(ctx, fmt.Sprintf("慢查询( cost_time: %v ,rows: %v , sql: %v )", elapsed, rows, sql))
	case g.level >= gorm_logger.Info:
		g.Logger.InfoCtx(ctx, fmt.Sprintf("SQL执行( cost_time: %v ,rows: %v , sql: %v )", elapsed, rows, sql))
	}
}
