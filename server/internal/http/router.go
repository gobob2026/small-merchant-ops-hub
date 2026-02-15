package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"small-merchant-ops-hub-server/internal/cache"
	"small-merchant-ops-hub-server/internal/config"
)

func NewRouter(db *gorm.DB, cacheStore cache.Store, cfg config.Config) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg.CORSAllowOrigin))

	router.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"ok":    false,
				"error": "database bridge unavailable",
			})
			return
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"ok":    false,
				"error": "database ping failed",
			})
			return
		}
		if err := cacheStore.Ping(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"ok":    false,
				"error": "cache ping failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":    true,
			"env":   cfg.Env,
			"db":    cfg.DatabaseDriver(),
			"cache": cfg.CacheMode,
		})
	})

	registerMerchantRoutes(router, db, cacheStore)

	return router
}

func corsMiddleware(allowOrigin string) gin.HandlerFunc {
	if allowOrigin == "" {
		allowOrigin = "*"
	}

	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
