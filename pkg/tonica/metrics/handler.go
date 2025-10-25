package metrics

import (
	"net/http"
	"runtime"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// GetHandler creates a new HTTP handler that serves metrics collected by the provided metrics manager to the '/metrics' route.
func GetHandler(m Manager, router *gin.Engine) *gin.Engine {
	handler := func() http.Handler {
		var stats runtime.MemStats

		runtime.ReadMemStats(&stats)

		m.SetGauge("app_go_routines", float64(runtime.NumGoroutine()))
		m.SetGauge("app_sys_memory_alloc", float64(stats.Alloc))
		m.SetGauge("app_sys_total_alloc", float64(stats.TotalAlloc))
		m.SetGauge("app_go_numGC", float64(stats.NumGC))
		m.SetGauge("app_go_sys", float64(stats.Sys))

		return promhttp.Handler()
	}
	router.GET("/metrics", gin.WrapH(handler()))
	pprof.Register(router)
	return router
}
