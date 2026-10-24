package server

import (
	"context"
	"net/http"
	"time"

	"biz-credit-ai/internal/auth"
	"biz-credit-ai/internal/config"
	"biz-credit-ai/internal/handler"
	"biz-credit-ai/internal/handlers"
	"biz-credit-ai/internal/middleware"
	"biz-credit-ai/internal/models"
	"biz-credit-ai/internal/scoring"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, tokens *auth.TokenManager, cfg ...*config.Config) *gin.Engine {
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

	// AI financial advisor routes
	var appCfg *config.Config
	if len(cfg) > 0 && cfg[0] != nil {
		appCfg = cfg[0]
	} else {
		appCfg = &config.Config{
			Server: config.ServerConfig{Addr: "0.0.0.0", Port: "8080"},
			AI: config.AIConfig{
				BaseURL: "http://10.154.0.1:8080/v1",
				Model:   "gemma4:12b",
				Timeout: 30,
			},
		}
	}
	h.Scoring = scoring.New(appCfg.ScoringBaseURL)
	client := r.Group("/api", middleware.AuthMiddleware(db, tokens), handlers.RequireRole(models.RoleClient))
	client.GET("/products", h.Products)
	client.POST("/applications", h.CreateApplication)
	client.GET("/applications/:id", h.GetApplication)
	client.POST("/applications/:id/submit", h.SubmitApplication)
	client.POST("/applications/:id/counter-offer", h.CounterOffer)
	client.POST("/applications/:id/documents", h.UploadDocuments)
	client.POST("/applications/:id/sign", h.SignApplication)
	officer := r.Group("/api/officer", middleware.AuthMiddleware(db, tokens), handlers.RequireRole(models.RoleOfficer))
	officer.GET("/applications", h.OfficerApplications)
	officer.POST("/applications/:id/decision", h.OfficerDecision)
	r.GET("/api/applications/:id/documents/:doc_id", middleware.AuthMiddleware(db, tokens), h.DownloadDocument)
	handler.RegisterRoutes(r, appCfg)

	r.NoRoute(func(c *gin.Context) { c.JSON(http.StatusNotFound, gin.H{"error": "Маршрут не найден"}) })
	return r
}
