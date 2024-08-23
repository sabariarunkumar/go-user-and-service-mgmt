package role

import (
	"userservice/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler for service management.
type Handler struct {
	operations models.RoleOperations
}

// NewHandler initializes role handler context with desired parameters.
func NewHandler(log *zap.SugaredLogger, db *gorm.DB) *Handler {
	return &Handler{operations: newOperations(db, log)}
}

// RegisterRoutes has sent of route
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {

	// Authorized routes for all user roles.
	router.GET("/roles", h.fetchUserRoles)
}
