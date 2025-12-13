package perfomance

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartPrometheusInGoroutine 独立启动一个goroutine部署Prometheus
func StartPrometheusInGoroutine(addr string) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:    addr,
			Handler: mux,
		}

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic("Error starting Prometheus server: " + err.Error())
		}
	}()
}

// StartPrometheusInGoroutineWithContext 独立启动一个goroutine部署Prometheus，支持上下文控制
func StartPrometheusInGoroutineWithContext(ctx context.Context, addr string) {
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
				panic("Error shutting down Prometheus server: " + err.Error())
			}
		}()

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic("Error starting Prometheus server: " + err.Error())
		}
	}()
}

// RegisterPrometheusToGinEngine 将Prometheus路由注册到已有的Gin服务
func RegisterPrometheusToGinEngine(engine *gin.Engine) {
	// 注册Prometheus路由到Gin引擎
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
