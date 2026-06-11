package log

import (
	"context"
)

type LogInterface interface {
	WriteLog(ctx context.Context, msg string, keysAndValues ...interface{})
	FatalLog(ctx context.Context, msg string, keysAndValues ...interface{})
}

// LoggerInterface 定义常规日志记录器接口。
type LoggerInterface interface {
	DebugCtx(ctx context.Context, msg string, keysAndValues ...interface{})
	InfoCtx(ctx context.Context, msg string, keysAndValues ...interface{})
	WarnCtx(ctx context.Context, msg string, keysAndValues ...interface{})
	ErrorCtx(ctx context.Context, msg string, keysAndValues ...interface{})
	FatalCtx(ctx context.Context, msg string, keysAndValues ...interface{})
}
