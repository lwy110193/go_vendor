package tracer

import (
	"context"
	stdlog "log"

	mylog "gitee.com/qq1101931365/go_verdor/log"

	otlptrace "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otlptracehttp "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/zipkin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// LoggerExporter 创建一个自定义的span导出器，使用指定的logger类型
// serviceName: 服务名称
// loggerType: 指定使用的logger类型
// logger: 当loggerType为LoggerTypeLocal时，必须传入有效的local log实例
// 返回: 自定义的span导出器
func LoggerExporter(loggerType mylog.LoggerType, logger ...*mylog.Logger) sdktrace.SpanExporter {
	var exporter *mylog.CustomExporter
	var err error

	// 根据logger类型创建不同的导出器
	switch loggerType {
	case mylog.LoggerTypeLocal:
		// 使用本地log库
		if len(logger) == 0 || logger[0] == nil {
			panic("loggerType为LoggerTypeLocal时，必须传入有效的local log实例")
		}
		exporter, err = mylog.NewCustomExporter(mylog.WithLocalLogger(logger[0]))
	case mylog.LoggerTypeStandard:
		// 使用标准库log
		exporter, err = mylog.NewCustomExporter(mylog.WithStandardLogger(stdlog.Default()))
	case mylog.LoggerTypeNoop:
		// 使用非输出logger
		exporter, err = mylog.NewCustomExporter(mylog.WithNoopLogger())
	default:
		stdlog.Fatalf("不支持的logger类型: %v", loggerType)
	}

	if err != nil {
		stdlog.Fatalf("创建自定义span导出器失败: %v", err)
	}

	return exporter
}

// JaegerExporter 创建一个OTLP span导出器（替代已弃用的Jaeger导出器）
// jaegerEndpoint: OTLP endpoint（如http://localhost:4318/v1/traces）
// 返回: OTLP span导出器
func JaegerExporter(jaegerEndpoint string) sdktrace.SpanExporter {
	// 1. 配置 OTLP HTTP 客户端
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpointURL(jaegerEndpoint),
	)

	// 2. 创建 OTLP 导出器
	exporter, err := otlptrace.New(
		context.Background(),
		client,
	)
	if err != nil {
		stdlog.Fatalf("创建OTLP span导出器失败: %v", err)
	}

	return exporter
}

// ZipkinExporter 创建一个zipkin span导出器
// serviceName: 服务名称
// zipkinEndpoint: zipkin endpoint
// 返回: zipkin span导出器
func ZipkinExporter(zipkinEndpoint string) sdktrace.SpanExporter {
	// 创建zipkin导出器
	exporter, err := zipkin.New(
		zipkinEndpoint,
	)
	if err != nil {
		stdlog.Fatalf("创建zipkin span导出器失败: %v", err)
	}

	return exporter
}
