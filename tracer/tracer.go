package tracer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"

	"strings"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracerWithLogger 初始化tracer，使用日志记录器，支持三种logger类型
// loggerType: 指定使用的logger类型
// logger: 当loggerType为LoggerTypeLocal时，必须传入有效的local log实例
// 返回: 清理函数
func InitTracer(serviceName string, exporter sdktrace.SpanExporter) func() {
	// 创建trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	// 设置全局trace provider
	otel.SetTracerProvider(tp)

	// 返回清理函数
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("关闭tracer provider失败: %v", err)
		}
	}
}

// NewTraceSpan 创建一个新的span
// ctx: 父上下文
// tracerName: tracer名称
// spanName: span名称
// 返回: 新的上下文、span实例
func NewTraceSpan(ctx context.Context, tracerName, spanName string) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, spanName)

	return ctx, span
}

// NewTrace 创建一个新的tracer
// ctx: 父上下文
// tracerName: tracer名称
// 返回: 新的tracer实例
func NewTrace(ctx context.Context, tracerName string) trace.Tracer {
	return otel.Tracer(tracerName)
}

// NewSpan 创建一个新的span
// ctx: 父上下文
// tracer: tracer实例
// spanName: span名称
// 返回: 新的上下文、span实例
func NewSpan(ctx context.Context, tracer trace.Tracer, spanName string) (context.Context, trace.Span) {
	return tracer.Start(ctx, spanName)
}

// NewSpanWithCtx 创建一个新的span，使用上下文中的traceID
// ctx: 父上下文
// spanName: span名称
// ctxTraceIdKey: 上下文key，用于存储traceID
// 返回: 新的上下文、span实例
func NewSpanWithCtx(ctx context.Context, traceName, spanName, ctxTraceIdKey string) (context.Context, trace.Span) {
	var traceID string
	if ctx.Value(ctxTraceIdKey) != nil {
		traceID = ctx.Value(ctxTraceIdKey).(string)
		if len(traceID) != 32 {
			traceID = strings.ReplaceAll(uuid.New().String(), "-", "")
		}
	} else {
		traceID = strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	tracer := otel.Tracer(traceName)
	id, _ := trace.TraceIDFromHex(traceID)
	sid, _ := trace.SpanIDFromHex(strings.ReplaceAll(uuid.New().String(), "-", "")[:16])

	// 创建新的 SpanContext
	newConfig := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    id,   // 16字节数组
		SpanID:     sid,  // 8字节数组
		Remote:     true, // 标记为远程上下文
		TraceFlags: trace.FlagsSampled,
		TraceState: trace.TraceState{},
	})

	// 创建关联新 TraceID 的 Span
	ctx = trace.ContextWithRemoteSpanContext(ctx, newConfig)
	_, span := tracer.Start(ctx, spanName)
	// defer span.End()

	return ctx, span
}

// GinTraceMiddleware 是一个Gin中间件，用于添加OpenTelemetry trace 111 222
func GinTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ginCtx := c.Request.Context()
		tracer := otel.Tracer("__TRACE__" + c.FullPath())
		spanName := "__SPAN__" + c.FullPath()
		ctx, span := tracer.Start(ginCtx, spanName)
		defer span.End()
		c.Request = c.Request.WithContext(ctx)

		span.SetAttributes(semconv.HTTPRouteKey.String(c.FullPath()))
		span.SetAttributes(attribute.String("path.params", c.Request.URL.Query().Encode()))
		span.SetAttributes(semconv.HTTPMethodKey.String(c.Request.Method))
		span.SetAttributes(semconv.HTTPHostKey.String(c.Request.Host))
		span.SetAttributes(semconv.HTTPURLKey.String(c.Request.URL.String()))
		span.SetAttributes(semconv.HTTPUserAgentKey.String(c.Request.UserAgent()))
		span.SetAttributes(semconv.HTTPRequestContentLengthKey.Int64(c.Request.ContentLength))

		if c.Request.Method == "POST" {
			span.SetAttributes(attribute.String("content_type", c.ContentType()))
			var params string
			if strings.Contains(c.ContentType(), "multipart/form-data") {
				_ = c.Request.ParseMultipartForm(32 << 20)
				for k, v := range c.Request.PostForm {
					if _, exists := c.Request.MultipartForm.File[k]; !exists {
						if len(v) > 1 {
							params += fmt.Sprintf("%s=%s&", k, fmt.Sprintf("[%s]", strings.Join(v, ",")))
						} else if len(v) == 1 {
							params += fmt.Sprintf("%s=%s&", k, v[0])
						} else {
							params += fmt.Sprintf("%s=&", k)
						}
					}
				}
				params = params[:len(params)-1]
			} else {
				rawData, err := c.GetRawData() // 读取请求体，注意：读取后会关闭Body，后续无法读取
				if err == nil && len(rawData) > 0 {
					params = string(rawData)
				}
				c.Request.Body = io.NopCloser(bytes.NewBuffer(rawData)) // 重置请求体，以便后续处理
			}
			span.SetAttributes(attribute.String("post.params", string(params)))
		}

		c.Set("tracer", tracer)
		c.Set("span", span)
		c.Set("span_ctx", ctx)
		c.Set("traceID", span.SpanContext().TraceID().String())
		c.Set("spanID", span.SpanContext().SpanID().String())

		c.Next()

		status := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCodeKey.String(strconv.Itoa(status)))
		if len(c.Errors) > 0 {
			span.SetAttributes(attribute.String("gin.errors", c.Errors.String()))
		}
	}
}
