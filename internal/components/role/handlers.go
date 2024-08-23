package role

import (
	"net/http"
	appErrors "userservice/internal/errors"
	"userservice/internal/utils"

	"github.com/gin-gonic/gin"
)

// fetchUserRoles respond with pre-configured user roles
func (h *Handler) fetchUserRoles(c *gin.Context) {
	roles, err := h.operations.FetchRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	c.JSON(http.StatusOK, roles)
}
