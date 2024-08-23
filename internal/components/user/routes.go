package user

import (
	"userservice/internal/configs"
	"userservice/internal/middleware"
	"userservice/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler for user management.
type Handler struct {
	runtimeConfig *configs.Config
	operations    models.UserOperations
}

// NewHandler initializes user handler context with desired parameters.
func NewHandler(log *zap.SugaredLogger, config *configs.Config, db *gorm.DB) *Handler {
	return &Handler{runtimeConfig: config, operations: newOperations(db, log)}
}

// RegisterRoutes has sent of route endpoints categorized as per authz roles using middleware.
func (h *Handler) RegisterRoutes(routers *gin.RouterGroup) {

	// Authorized routes for all user roles.
	routers.POST("/login", h.login)
	routers.PUT("/user/self/password", h.changeUserPassword)

	// Authorized routes for advanced, and admin users.
	advancedAndAdminUserRoutes := routers.Group("/")
	advancedAndAdminUserRoutes.Use(middleware.AuthzRoles(models.RoleAdvanced, models.RoleAdmin))
	{
		advancedAndAdminUserRoutes.GET("/user/:id", h.getUserByID)
		advancedAndAdminUserRoutes.GET("/users", h.fetchUsers)
	}

	// Authorized routes only for admin roles.
	adminUserOnlyRoutes := routers.Group("/")
	adminUserOnlyRoutes.Use(middleware.AuthzRoles(models.RoleAdmin))
	{
		adminUserOnlyRoutes.POST("/user", h.addUser)
		adminUserOnlyRoutes.PUT("/user/:id", h.updateUser)
		adminUserOnlyRoutes.DELETE("/user/:id", h.deleteUser)
	}

}
