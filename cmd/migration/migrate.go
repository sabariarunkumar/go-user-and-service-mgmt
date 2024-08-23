package migration

import (
	"fmt"
	"userservice/internal/auth"
	"userservice/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	adminUserName     = "admin"
	adminUserRole     = "admin"
	adminUserEmail    = "admin@mgmtportal.com"
	adminUserPassword = "admin123"
)

var defaultRoles = func() map[string]string {
	return map[string]string{
		"basic":    "View service(s) and their version(s)",
		"advanced": "View/Modify service(s) and view user(s) in systems",
		"admin":    "View/Modify service(s) and user(s) in systems",
	}
}

// MigrateDBEntities migrate db tables, initializes admin users, roles and views
func MigrateDBEntities(log *zap.SugaredLogger, db *gorm.DB) error {
	if err := db.AutoMigrate(&models.User{}); err != nil {
		return fmt.Errorf("failed to migrate Users table: %v", err)
	}
	log.Info("Successfully Migrated User table")
	if err := db.AutoMigrate(&models.Service{}); err != nil {
		return fmt.Errorf("failed to migrate Service table: %+v", err)
	}
	log.Info("Successfully Migrated Service table")
	if err := db.AutoMigrate(&models.ServiceVersion{}); err != nil {
		return fmt.Errorf("failed to migrate Service Version table: %+v", err)
	}
	log.Info("Successfully Migrated Service Version table")
	if err := db.AutoMigrate(&models.UserRole{}); err != nil {
		return fmt.Errorf("failed to migrate UserRole table: %+v", err)
	}
	nameSortedServiceView := `
    CREATE MATERIALIZED VIEW IF NOT EXISTS name_sorted_service AS
    SELECT *
    FROM service
    ORDER BY name;`

	if err := db.Exec(nameSortedServiceView).Error; err != nil {
		return fmt.Errorf("failed to create name_sorted_service view materialized view: %v", err)
	}
	log.Info("materialized view name_sorted_service created successfully.")

	return nil
}

// InitDBEntities initializes admin users, roles.
func InitDBEntities(log *zap.SugaredLogger, db *gorm.DB) error {
	// view represents the services sorted in ascending order

	passHash, err := auth.GeneratePasswordHash(adminUserPassword)
	if err != nil {
		return fmt.Errorf("internal error while generating a password hash: %v", err)
	}
	var userWithSameEmail int64 = 0
	if gormErr := db.Model(&models.User{}).Where("email = ?", adminUserEmail).
		Count(&userWithSameEmail).Error; gormErr != nil {
		return fmt.Errorf("internal error while adding admin user: %v", gormErr)
	}
	if userWithSameEmail == 0 {
		adminUser := models.User{Name: adminUserName, Email: adminUserEmail, Role: adminUserRole,
			PasswordHash: passHash, IsTemporaryPassword: true}
		if gormErr := db.Model(&models.User{}).Create(&adminUser).Error; gormErr != nil {
			return fmt.Errorf("failed to create admin user: %v", gormErr)
		}
		log.Infof("%s user configured successfully;", adminUserName)
	}

	for role, description := range defaultRoles() {
		userRole := models.UserRole{Name: role, Description: description}
		var roleConfigured int64 = 0
		if gormErr := db.Model(&models.UserRole{}).Where("name = ?", role).
			Count(&roleConfigured).Error; gormErr != nil {
			return fmt.Errorf("internal error while configuring role %s user: %v", role, gormErr)
		}
		if roleConfigured == 0 {
			if gormErr := db.Model(&models.UserRole{}).Create(&userRole).Error; gormErr != nil {
				return fmt.Errorf("failed to create role %s: %v", role, gormErr)
			}
			log.Infof("Role %s configured successfully", role)
		}
	}
	return nil
}
