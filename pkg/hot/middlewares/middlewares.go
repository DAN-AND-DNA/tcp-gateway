package middlewares

import (
	"fmt"
	"gateway/pkg/metric"
	"gateway/pkg/utils"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// Custom Panic Handling Middleware
func CustomRecovery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				utils.AlertAuto(fmt.Sprintf("private http service panic, client ip: %s err: %v stack: %s", ctx.RemoteIP(), err, string(debug.Stack())))

				// Return Custom Error Response
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"code":    1,
					"message": "panic",
				})
			}
		}()
		ctx.Next()
	}
}

func LogMiddle() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		metric.CountPrivateHTTPRequest.Add(1)
		start := time.Now()
		ctx.Next()
		cost := time.Now().Sub(start).Milliseconds()

		if ctx.Request.URL != nil {
			path := ctx.Request.URL.Path

			// Statistics
			metric.P99PrivateHTTPRequestLatency.In(path, cost)

			// Slow Query Alert
			if cost >= 4000 {
				utils.AlertAuto(fmt.Sprintf("private http service slow request, client ip: %s path: %s cost: %d ms", ctx.ClientIP(), path, cost))
			}
		}
	}
}
