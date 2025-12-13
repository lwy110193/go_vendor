package log

import (
	"context"
	"time"

	gorm_logger "gorm.io/gorm/logger"
)

// 对外使用的方法 start

// Debugw 记录调试级别结构化日志
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.Debugw(msg, keysAndValues...)
}

// Infow 记录信息级别结构化日志
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.Infow(msg, keysAndValues...)
}

// Warningw 记录警告级别结构化日志
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.Warnw(msg, keysAndValues...)
}

// Errorw 记录错误级别结构化日志
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.Errorw(msg, keysAndValues...)
}

// Fatalw 记录致命级别结构化日志，并终止程序
func (l *Logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.Fatalw(msg, keysAndValues...)
}

// 对外使用的方法 end

// gorm 日志接口实现 start

// GORMLogger GORM 日志接口实现
type GORMLogger struct {
	Logger *Logger
	level  gorm_logger.LogLevel
}

// AsGORMLogger 将普通日志记录器转换为 GORM 日志记录器
func (l *Logger) AsGORMLogger() *GORMLogger {
	return NewGORMLogger(l)
}

// NewGORMLogger 创建一个新的 GORM 日志记录器
func NewGORMLogger(logger *Logger) *GORMLogger {
	return &GORMLogger{
		Logger: logger,
		level:  gorm_logger.Info, // 默认为 Info 级别
	}
}

// LogMode 设置日志级别并返回新的日志记录器
func (g *GORMLogger) LogMode(level gorm_logger.LogLevel) gorm_logger.Interface {
	newLogger := *g
	newLogger.level = level
	return &newLogger
}

// Info 记录信息级别日志
func (g *GORMLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if g.level >= gorm_logger.Info {
		g.Logger.Infowc(ctx, msg, data...)
	}
}

// Warn 记录警告级别日志
func (g *GORMLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if g.level >= gorm_logger.Warn {
		g.Logger.Warnwc(ctx, msg, data...)
	}
}

// Error 记录错误级别日志
func (g *GORMLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if g.level >= gorm_logger.Error {
		g.Logger.Errorwc(ctx, msg, data...)
	}
}

// Trace 记录 SQL 执行信息
func (g *GORMLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if g.level <= gorm_logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// 从 context 中提取有用的信息
	fields := []interface{}{
		"elapsed", elapsed,
		"rows", rows,
		"sql", sql,
	}

	switch {
	case err != nil && g.level >= gorm_logger.Error:
		g.Logger.Errorwc(ctx, "SQL执行错误", append(fields, "error", err)...)
	case elapsed > 200*time.Millisecond && g.level >= gorm_logger.Warn:
		g.Logger.Warnwc(ctx, "慢查询", fields...)
	case g.level >= gorm_logger.Info:
		g.Logger.Infowc(ctx, "SQL执行", fields...)
	}
}

// gorm 日志接口实现 end
