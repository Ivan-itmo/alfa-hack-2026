package handler

import (
	"github.com/gin-gonic/gin"

	"biz-credit-ai/internal/config"
	"biz-credit-ai/internal/service"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	svc := service.NewAdvisorService(cfg)
	h := NewAdvisorHandler(svc, cfg)

	api := r.Group("/api/v1")
	{
		advice := api.Group("/advice")
		{
			advice.POST("/compare", h.CompareProducts)
			advice.POST("/contextual", h.GetContextualAdvice)
			advice.POST("/explain-decision", h.ExplainBankDecision)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
