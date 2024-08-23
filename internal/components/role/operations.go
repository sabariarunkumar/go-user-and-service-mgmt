package role

import (
	appErrors "userservice/internal/errors"
	"userservice/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// operations...
type operations struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

// newOperations initializes operations with necessary configs
func newOperations(db *gorm.DB, log *zap.SugaredLogger) *operations {
	return &operations{db: db, log: log}
}

// FetchRoles fetches roles from DB
func (ops *operations) FetchRoles() (userRoles []models.UserRole, returnErr error) {
	err := ops.db.Find(&userRoles).Error
	if err != nil {
		ops.log.Errorf("Failed to fetch user roles: %+v", err)
		return nil, appErrors.ErrInternal
	}
	return
}
