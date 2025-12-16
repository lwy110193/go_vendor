package log

import (
	"context"
)

type LogInterface interface {
	WriteLog(ctx context.Context, msg string, keysAndValues ...interface{})
	FatalLog(ctx context.Context, msg string, keysAndValues ...interface{})
}
