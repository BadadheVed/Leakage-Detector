package route

import (
	"github.com/BadadheVed/leakage-detector/setup"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, setup *setup.Config) {
	// Health check route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/repo/:url", func(c *gin.Context) {
		ScanRepo(c, setup)
	})
}
