package router

import (
	"github.com/gin-gonic/gin"

	"biz-credit-ai/internal/config"
)

type Router struct {
	engine *gin.Engine
	cfg    *config.Config
}

func New(engine *gin.Engine, cfg *config.Config) *Router {
	return &Router{
		engine: engine,
		cfg:    cfg,
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}
