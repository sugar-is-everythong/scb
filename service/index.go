package service

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// GinService implements a Wails service that uses Gin for HTTP handling
type GinService struct {
	ginEngine *gin.Engine
}

// NewGinService creates a new GinService instance
func NewGinService() *GinService {
	ginEngine := gin.New()
	ginEngine.Use(gin.Recovery())
	gin.SetMode(gin.ReleaseMode)

	service := &GinService{ginEngine: ginEngine}
	service.setupRoutes()

	return service
}

// ServiceName returns the name of the service
func (s *GinService) ServiceName() string {
	return "Gin API Service"
}

// ServiceStartup is called when the service starts
func (s *GinService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return nil
}

// ServiceShutdown is called when the service shuts down.
func (s *GinService) ServiceShutdown() error {
	return nil
}

// ServeHTTP implements the http.Handler interface
func (s *GinService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.ginEngine.ServeHTTP(w, r)
}
