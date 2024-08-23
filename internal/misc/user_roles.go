package misc

import (
	"fmt"
	"userservice/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// app(s) require high performant fast concurrent reads, hence opting for Map.
	Roles = make(map[string]string)
)

// LoadUserRoles records pre-configured user roles from database into memory.
func LoadUserRoles(log *zap.SugaredLogger, db *gorm.DB) error {
	var userRoles []models.UserRole
	err := db.Find(&userRoles).Error
	if err != nil {
		return fmt.Errorf("unable to fetch user roles from DB: %+v", err)
	}
	if len(userRoles) == 0 {
		return fmt.Errorf("no User roles configured; Run Migration")
	}
	for _, userRole := range userRoles {
		Roles[userRole.Name] = userRole.Description
	}
	return nil
}
