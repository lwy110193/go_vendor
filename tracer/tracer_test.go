package tracer

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"

	"strings"

	"github.com/google/uuid"
	mylog "github.com/lwy110193/go_vendor/log"
)

// TestOtlpWithJaeger 测试OTLPExample函数
func TestOtlpWithJaeger(t *testing.T) {
	end_func := InitTracer("OTLP1", getJaegerExporter())
	defer end_func()

	ctx1 := context.WithValue(context.Background(), "id", strings.ReplaceAll(uuid.New().String(), "-", ""))
	ctx, span := NewSpanWithCtx(ctx1, "TestOtlpWithJaeger", "main-operation", "id")
	// ctx, span := NewTraceSpan(context.Background(), "TestOtlpWithJaeger", "main-operation")
	defer span.End()
	span.AddEvent("main-operation-start")
	span.SetAttributes(
		attribute.String("key1", "value1"),
	)
	childWork(ctx)
	span.AddEvent("main-operation-end")
	localLogger.Info("普通日志")
	localLogger.Infowc(ctx, "日志+traceID+spanID")
}

// TestOtlpWithLogger 测试OTLPExample函数
func TestOtlpWithLogger(t *testing.T) {
	end_func := InitTracer("OTLPExample", getLoggerExporter())
	defer end_func()

	ctx, span := NewTraceSpan(context.Background(), "TestOtlpWithLogger", "main-operation")
	defer span.End()
	span.AddEvent("main-operation-start")
	span.SetAttributes(
		attribute.String("key1", "value1"),
	)

	childWork(ctx)
	span.AddEvent("main-operation-end")
}

// TestOtlpLogger 测试OTLP 日志记录
func TestOtlpLogger(t *testing.T) {
	end_func := InitTracer("OTLPExample", nil)
	defer end_func()

	ctx, span := NewTraceSpan(context.Background(), "TestOtlpWithLogger", "main-operation")
	defer span.End()

	localLogger.Info("普通日志")
	localLogger.Infowc(ctx, "日志+traceID+spanID")
}

func childWork(ctx context.Context) {
	_, span := NewTraceSpan(ctx, "doWork", "doWork")
	span.AddEvent("childWork-start")
	span.SetAttributes(
		attribute.String("key2", "value2"),
	)
	defer span.End()
	// 子任务结束事件
	span.AddEvent("childWork-end")
}

var (
	localLogger *mylog.Logger
)

func init() {
	localLogger, _ = mylog.New(mylog.Config{
		Level:         mylog.INFO,
		Development:   false,
		OutputDir:     "./logs",
		Filename:      "custom_trace.log",
		MaxSize:       100,
		MaxAge:        7,
		ByDate:        true,
		Encoding:      "json",
		StdoutEnable:  true,
		FileOutEnable: true,
	})
}

func getLoggerExporter() tracesdk.SpanExporter {

	exporter := LoggerExporter(mylog.LoggerTypeLocal, localLogger)
	return exporter
}

func getJaegerExporter() tracesdk.SpanExporter {
	exporter := JaegerExporter("http://192.168.3.42:4318/v1/traces")
	return exporter
}
