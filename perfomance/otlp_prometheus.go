package perfomance

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// 全局变量用于存储 MeterProvider 和 Meter
var (
	meterProvider *sdkmetric.MeterProvider
	meter         metric.Meter
)

// InitOpenTelemetryPrometheus 初始化 OpenTelemetry Prometheus 导出器
func InitOpenTelemetryPrometheus(name string) error {
	// 创建 Prometheus 导出器
	exporter, err := prometheus.New()
	if err != nil {
		return fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// 创建 MeterProvider
	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)

	// 设置全局 MeterProvider
	otel.SetMeterProvider(meterProvider)

	// 获取全局 Meter
	meter = otel.Meter(name)

	return nil
}

// GetMeter 返回全局 Meter 实例
func GetMeter() metric.Meter {
	return meter
}

// Shutdown 关闭 MeterProvider，确保所有数据都被导出
func Shutdown(ctx context.Context) error {
	if meterProvider != nil {
		return meterProvider.Shutdown(ctx)
	}
	return nil
}

// StartPrometheusWithOpenTelemetry 启动一个独立的 HTTP 服务器来暴露 Prometheus 指标
func StartPrometheusWithOpenTelemetry(addr string) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:    addr,
			Handler: mux,
		}

		log.Printf("Starting Prometheus server with OpenTelemetry metrics at %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting Prometheus server: %v", err)
		}
	}()
}

// StartPrometheusWithOpenTelemetryAndContext 启动一个支持上下文控制的 HTTP 服务器来暴露 Prometheus 指标
func StartPrometheusWithOpenTelemetryAndContext(ctx context.Context, addr string) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:    addr,
			Handler: mux,
		}

		// 监听关闭信号
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down Prometheus server: %v", err)
			}

			// 关闭 MeterProvider
			if err := Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down MeterProvider: %v", err)
			}
		}()

		log.Printf("Starting Prometheus server with OpenTelemetry metrics at %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting Prometheus server: %v", err)
		}
	}()
}

// RegisterPrometheusToGinEngineWithOpenTelemetry 将 Prometheus 路由注册到已有的 Gin 服务
func RegisterPrometheusToGinEngineWithOpenTelemetry(engine *gin.Engine) {
	// 注册 Prometheus 路由到 Gin 引擎
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// 示例：创建 Counter 指标
func CreateCounterExample() (metric.Int64Counter, error) {
	counter, err := meter.Int64Counter(
		"app_requests_total",
		metric.WithDescription("Total number of application requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create counter: %w", err)
	}
	return counter, nil
}

// 示例：创建 Histogram 指标
func CreateHistogramExample() (metric.Float64Histogram, error) {
	histogram, err := meter.Float64Histogram(
		"app_request_duration_seconds",
		metric.WithDescription("Application request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create histogram: %w", err)
	}
	return histogram, nil
}

// 示例：创建 UpDownCounter 指标
func CreateUpDownCounterExample() (metric.Int64UpDownCounter, error) {
	upDownCounter, err := meter.Int64UpDownCounter(
		"app_active_users",
		metric.WithDescription("Number of active users"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create up-down counter: %w", err)
	}
	return upDownCounter, nil
}

// 实例：展示如何使用 OpenTelemetry Prometheus 导出器
func Example() {

	// 初始化 OpenTelemetry Prometheus 导出器
	if err := InitOpenTelemetryPrometheus("go_vendor_app"); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry Prometheus: %v", err)
	}

	// 启动 Prometheus 服务器来暴露指标
	StartPrometheusWithOpenTelemetry(":2112")

	// 创建示例指标
	counter, err := CreateCounterExample()
	if err != nil {
		log.Fatalf("Failed to create counter: %v", err)
	}

	histogram, err := CreateHistogramExample()
	if err != nil {
		log.Fatalf("Failed to create histogram: %v", err)
	}

	upDownCounter, err := CreateUpDownCounterExample()
	if err != nil {
		log.Fatalf("Failed to create up-down counter: %v", err)
	}

	// 模拟应用程序运行并记录指标
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		// 增加计数器
		counter.Add(ctx, 1)

		// 记录直方图值
		start := time.Now()
		time.Sleep(time.Millisecond * time.Duration(i%100)) // 模拟处理时间
		duration := time.Since(start).Seconds()
		histogram.Record(ctx, duration)

		// 更新上下计数器
		if i%2 == 0 {
			upDownCounter.Add(ctx, 1)
		} else {
			upDownCounter.Add(ctx, -1)
		}

		time.Sleep(time.Second)
	}

	// 保持应用程序运行
	select {}
}
