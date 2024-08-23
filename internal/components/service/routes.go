package service

import (
	"context"
	"sync"
	"userservice/internal/configs"
	"userservice/internal/middleware"
	"userservice/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler for service management.
type Handler struct {
	runtimeConfig *configs.Config
	operations    models.ServiceOperations
}

// NewHandler initializes service handler context with desired parameters.
func NewHandler(ctx context.Context, wg *sync.WaitGroup, log *zap.SugaredLogger, config *configs.Config, db *gorm.DB) *Handler {
	return &Handler{runtimeConfig: config, operations: newOperations(ctx, wg, db, log)}
}

// RegisterRoutes has sent of route endpoints categorized as per authz roles using middleware
func (h *Handler) RegisterRoutes(routers *gin.RouterGroup) {

	// Authorized routes for all user roles.
	routers.GET("/service/:id", h.getServiceByID)
	routers.GET("/services", h.fetchServices)
	routers.GET("/service/:id/version/:tag", h.getServiceVersion)
	routers.GET("/service/:id/versions", h.fetchServiceVersions)

	// Authorized routes for advanced, and admin users.
	advancedAndAdminRoutes := routers.Group("/")
	advancedAndAdminRoutes.Use(middleware.AuthzRoles(models.RoleAdvanced, models.RoleAdmin))
	{
		advancedAndAdminRoutes.POST("/service", h.addService)
		advancedAndAdminRoutes.PUT("/service/:id", h.updateService)
		advancedAndAdminRoutes.DELETE("/service/:id", h.deleteService)
		advancedAndAdminRoutes.POST("/service/:id/version", h.addServiceVersion)
		advancedAndAdminRoutes.PUT("/service/:id/version/:tag", h.updateServiceVersion)
		advancedAndAdminRoutes.DELETE("/service/:id/version/:tag", h.deleteServiceVersion)
	}

}
