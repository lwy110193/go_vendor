package log

import (
	"context"
	"encoding/json"
	stdlog "log"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// 自定义日志写入接口
// 支持不同的日志库实现
// 可以调用Printf或Info方法写入日志
type LogWriter interface {
	// WriteLog 写入日志，接收格式化的日志内容
	WriteLog(format string, args ...interface{})
}

// StandardLogWriter 标准库log的实现
type StandardLogWriter struct {
	logger *stdlog.Logger
}

// WriteLog 实现LogWriter接口，输出JSON格式日志
func (w *StandardLogWriter) WriteLog(format string, args ...interface{}) {
	// 创建日志对象
	logData := map[string]interface{}{
		"tracer_name": args[7], // 添加tracer名称
		"name":        args[0],
		"trace_id":    args[1],
		"span_id":     args[2],
		"p_trace_id":  args[3],
		"p_span_id":   args[4],
		"event_names": args[5],
		"resources":   args[6],
		// "time":        time.Now().Format("2006-01-02 15:04:05.000"), // 毫秒时间
	}

	// 转换为JSON
	jsonData, err := json.Marshal(logData)
	if err != nil {
		w.logger.Printf("JSON marshal error: %v, format: %s, args: %v", err, format, args)
		return
	}

	// 输出JSON
	w.logger.Println(string(jsonData))
}

// NewStandardLogWriter 创建标准库log的LogWriter实例
func NewStandardLogWriter(logger *stdlog.Logger) *StandardLogWriter {
	if logger == nil {
		logger = stdlog.Default()
	}
	return &StandardLogWriter{logger: logger}
}

// LocalLogWriter 本地log库的实现
type LocalLogWriter struct {
	logger *Logger
}

// WriteLog 实现LogWriter接口，输出JSON格式日志
func (w *LocalLogWriter) WriteLog(format string, args ...interface{}) {
	// 直接使用本地log库的Infow方法，传入具体的键值对
	w.logger.Infow("trace_span",
		"tracer_name", args[7], // 添加tracer名称
		"name", args[0],
		"trace_id", args[1],
		"span_id", args[2],
		"p_trace_id", args[3],
		"p_span_id", args[4],
		"event_names", args[5],
		"resources", args[6],
		// "time", time.Now().Format("2006-01-02 15:04:05.000"), // 毫秒时间
	)
}

// NewLocalLogWriter 创建本地log库的LogWriter实例
func NewLocalLogWriter(logger *Logger) *LocalLogWriter {
	return &LocalLogWriter{logger: logger}
}

// 自定义导出器配置
var (
	DefaultLogWriter = NewStandardLogWriter(nil)
)

// 自定义导出器
func NewCustomExporter(options ...CustomExporterOption) (*CustomExporter, error) {
	e := &CustomExporter{
		logWriter: DefaultLogWriter,
	}

	for _, opt := range options {
		opt(e)
	}

	return e, nil
}

// CustomExporter 是自定义的trace导出器
// 只输出指定的字段
// Name name，SpanContext.TraceID traceid，SpanContext.SpanID spanid，
// Parent.TraceID ptraceid，Parent.SpanID pspanid，Events.Name event_names []
// Resource.Key resource_key,Resource.Value.Value resource_value

type CustomExporter struct {
	logWriter LogWriter
}

// CustomExporterOption 是自定义导出器的选项
type CustomExporterOption func(*CustomExporter)

// WithStandardLogger 设置使用标准库log写入日志
func WithStandardLogger(logger *stdlog.Logger) CustomExporterOption {
	return func(e *CustomExporter) {
		e.logWriter = NewStandardLogWriter(logger)
	}
}

// WithLocalLogger 设置使用本地log库写入日志
func WithLocalLogger(logger *Logger) CustomExporterOption {
	return func(e *CustomExporter) {
		e.logWriter = NewLocalLogWriter(logger)
	}
}

// ExportSpans 实现trace.SpanExporter接口
func (e *CustomExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		// 获取Name
		name := span.Name()

		// 获取SpanContext信息
		spanCtx := span.SpanContext()
		traceID := spanCtx.TraceID().String()
		spanID := spanCtx.SpanID().String()

		// 获取Parent信息
		parentCtx := span.Parent()
		var parentTraceID, parentSpanID string
		if parentCtx.IsValid() {
			parentTraceID = parentCtx.TraceID().String()
			parentSpanID = parentCtx.SpanID().String()
		}

		// 获取Events.Name列表
		events := span.Events()
		eventNames := make([]string, len(events))
		for i, event := range events {
			eventNames[i] = event.Name
		}

		// 获取Resource信息
		res := span.Resource()
		attrs := res.Attributes()
		resources := make(map[string]string, len(attrs))
		for _, attr := range attrs {
			resources[string(attr.Key)] = attr.Value.AsString()
		}

		// 获取Tracer名称（Instrumentation Scope）
		tracerName := span.InstrumentationScope().Name

		// 使用LogWriter写入日志
		e.logWriter.WriteLog(
			"name=%s trace_id=%s span_id=%s p_trace_id=%s p_span_id=%s event_names=%v resources=%v tracer_name=%s",
			name, traceID, spanID, parentTraceID, parentSpanID, eventNames, resources, tracerName,
		)
	}

	return nil
}

// Shutdown 实现trace.SpanExporter接口
func (e *CustomExporter) Shutdown(ctx context.Context) error {
	return nil
}

// NoopLogWriter 是一个非输出的LogWriter实现
// 不实际写入任何日志
// 用于不需要输出的场景

type NoopLogWriter struct{}

// WriteLog 实现LogWriter接口，不输出任何日志
func (w *NoopLogWriter) WriteLog(format string, args ...interface{}) {
	// 空实现，不输出任何日志
}

// NewNoopLogWriter 创建一个非输出的LogWriter实例
func NewNoopLogWriter() *NoopLogWriter {
	return &NoopLogWriter{}
}

// WithNoopLogger 设置使用非输出日志
func WithNoopLogger() CustomExporterOption {
	return func(e *CustomExporter) {
		e.logWriter = NewNoopLogWriter()
	}
}

// LoggerType 定义支持的logger类型

type LoggerType int

const (
	// LoggerTypeLocal 使用本地log库
	LoggerTypeLocal LoggerType = iota
	// LoggerTypeStandard 使用标准库log
	LoggerTypeStandard
	// LoggerTypeNoop 使用非输出logger
	LoggerTypeNoop
)
