package server

import (
	"context"
	"net/http"
	"time"

	"alfa-credit/internal/auth"
	"alfa-credit/internal/handlers"
	"alfa-credit/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, tokens *auth.TokenManager) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	// Не доверяем произвольным X-Forwarded-For без явно настроенного прокси.
	_ = r.SetTrustedProxies(nil)
	r.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		pool, err := db.DB()
		if err != nil || pool.PingContext(ctx) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	h := &handlers.Handler{DB: db, Tokens: tokens}
	authRoutes := r.Group("/api/auth")
	authRoutes.POST("/register", h.Register)
	authRoutes.POST("/login", h.Login)
	dashboard := r.Group("/api/dashboard", middleware.AuthMiddleware(db, tokens))
	dashboard.GET("", h.Dashboard)
	dashboard.GET("/", h.Dashboard)
	r.NoRoute(func(c *gin.Context) { c.JSON(http.StatusNotFound, gin.H{"error": "Маршрут не найден"}) })
	return r
}
