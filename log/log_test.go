package log_test

import (
	"context"
	stdlog "log"
	"testing"

	"gitee.com/qq1101931365/go_verdor/log"
	trace "gitee.com/qq1101931365/go_verdor/tracer"
)

func TestLogger(t *testing.T) {
	// 创建本地log库的日志记录器
	logger, err := log.New(log.Config{
		StdoutEnable:  true,
		FileOutEnable: true,
		Level:         log.INFO,
		Development:   false,
		ErrorSperate:  true,
		OutputDir:     "./logs",
		Filename:      "custom_trace.log",
		MaxSize:       100,
		MaxAge:        7,
		ByDate:        true,
		Encoding:      "json",
	})
	if err != nil {
		stdlog.Fatal(err)
	}
	defer logger.Close()

	// ctx := context.WithValue(context.Background(), "tkey", "12345678901234567890123456789012")

	// 初始化tracer，使用本地log库
	shutdown := trace.InitTracer("test", nil)
	defer shutdown()

	ctx, span := trace.NewSpanWithCtx(context.WithValue(context.Background(), "id", "12345678901234567890123456789012"), "example-tracer", "tkey", "id")
	defer span.End()

	logger.Debugwc(ctx, "doWork debugwwwwwwwwwwwwww")
	logger.Errorwc(ctx, "doWork errorwwwwwwwwwwwwww")
	logger.Warnwc(ctx, "doWork warnwwwwwwwwwwwwww")
	logger.Debugwc(ctx, "doWork debugwwwwwwwwwwwwww")
	logger.Fatalwc(ctx, "doWork fatalwwwwwwwwwwwwww")

	logger.Infowc(ctx, "doWork endwwwwwwwwwwwwww")
}
