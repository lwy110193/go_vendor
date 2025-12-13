package perfomance

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
)

// StartPProfInGoroutine 独立启动一个goroutine部署pprof
func StartPProfInGoroutine(addr string) {
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			panic("Error starting pprof server: " + err.Error())
		}
	}()
}

// RegisterPProfToGinEngine 将pprof路由注册到已有的Gin服务
func RegisterPProfToGinEngine(engine *gin.Engine) {
	// 注册pprof路由到Gin引擎
	pprofGroup := engine.Group("/debug/pprof")
	{
		pprofGroup.GET("/", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/cmdline", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/profile", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.POST("/symbol", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/symbol", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/trace", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/allocs", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/block", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/goroutine", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/heap", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/mutex", gin.WrapF(http.DefaultServeMux.ServeHTTP))
		pprofGroup.GET("/threadcreate", gin.WrapF(http.DefaultServeMux.ServeHTTP))
	}
}
