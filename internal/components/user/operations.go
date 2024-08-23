package user

import (
	"errors"
	"strings"
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

// newOperations initializes user operation handler
func newOperations(db *gorm.DB, log *zap.SugaredLogger) *operations {
	return &operations{db: db, log: log}
}

// GetUserByEmail fetches user record in DB for the given email
func (ops *operations) GetUserByEmail(email string) (user *models.User, returnErr error) {
	user = new(models.User)
	gormErr := ops.db.Where("email = ?", email).First(user).Error
	if gormErr != nil {
		// set user to nil indicating caller about existence of valid user obj
		user = nil
		if errors.Is(gormErr, gorm.ErrRecordNotFound) {
			returnErr = gorm.ErrRecordNotFound
		} else {
			ops.log.Errorf("Failed to fetch user record by email %s : %v ", email, gormErr)
			returnErr = appErrors.ErrInternal
		}
	}
	return
}

// GetUser fetches user record in DB for the given id
func (ops *operations) GetUser(id uint) (user *models.User, returnErr error) {
	user = new(models.User)
	gormErr := ops.db.Where("id = ?", id).First(user).Error
	if gormErr != nil {
		// set user to nil indicating caller about existence of valid user obj
		user = nil
		if errors.Is(gormErr, gorm.ErrRecordNotFound) {
			returnErr = gorm.ErrRecordNotFound
		} else {
			ops.log.Errorf("Failed to fetch user record by id %d : %v ", id, gormErr)
			returnErr = appErrors.ErrInternal
		}
	}
	return
}

// CreateUser creates user record in DB  with necessary metadata.
// Since user creation happens seldom, we have additional DB call
// to check if record exist with same email rather than waiting for DB to report uniqueKey constrain.
// We still need to handle duplicate record constrain gracefully if create request happens at once
func (ops *operations) CreateUser(name string, email string, role string, passwordHash string) error {

	newUser := models.User{Name: name, Email: email, Role: role, PasswordHash: passwordHash, IsTemporaryPassword: true}
	var userWithSameEmail int64 = 0
	if gormErr := ops.db.Model(&models.User{}).Where("email = ?", email).Count(&userWithSameEmail).Error; gormErr != nil {
		return appErrors.ErrInternal
	}
	if userWithSameEmail == 1 {
		return appErrors.ErrUserWithSameEmailAlreadyExists
	}

	if gormErr := ops.db.Model(&models.User{}).Create(&newUser).Error; gormErr != nil {
		if strings.Contains(gormErr.Error(), appErrors.ErrUniqueKeyConstrainViolation.Error()) {
			return appErrors.ErrUserWithSameEmailAlreadyExists
		}
		ops.log.Errorf("Failed to create user with email %s: %v", email, gormErr)
		return appErrors.ErrInternal
	}
	return nil
}

// UpdateUser updates existing user record in DB with necessary metadata.
// Since user update happens seldom, we have additional DB call
// to verify if there is already a record [associated with other user] in the system that contains
// the email address specified in an update request rather than waiting for DB to report uniqueKey constrain.
// We still need to handle duplicate record constrain gracefully in distributed/concurrent environment.
func (ops *operations) UpdateUser(id uint, name string, email string, role string) (*models.User, error) {
	var userCountByID int64
	if err := ops.db.Model(&models.User{}).Where("id = ?", id).Count(&userCountByID).Error; err != nil {
		ops.log.Errorf("Failed to determine if a user with id %d is already registered: %v ", id, err)
		return nil, appErrors.ErrInternal
	}
	if userCountByID == 0 {
		return nil, appErrors.ErrUserDoesNotExist
	}

	var userWithSameEmail int64 = 0
	if gormErr := ops.db.Model(&models.User{}).Where("email = ? and id != ?", email, id).
		Count(&userWithSameEmail).Error; gormErr != nil {
		return nil, appErrors.ErrInternal
	}
	if userWithSameEmail == 1 {
		return nil, appErrors.ErrUserWithSameEmailAlreadyExists
	}

	userToUpdate := &models.User{Name: name, Email: email, Role: role, DBModel: models.DBModel{ID: id}}
	if gormErr := ops.db.Model(&models.User{}).Where("id = ?", id).Updates(userToUpdate).Error; gormErr != nil {
		if strings.Contains(gormErr.Error(), appErrors.ErrUniqueKeyConstrainViolation.Error()) {
			return nil, appErrors.ErrUserWithSameEmailAlreadyExists
		} else if errors.Is(gormErr, gorm.ErrRecordNotFound) {
			// Handle any concurrent deletion as well
			return nil, appErrors.ErrUserDoesNotExist
		}
		ops.log.Errorf("Failed to update user with email %s: %v", email, gormErr)
		return nil, appErrors.ErrInternal
	}
	return userToUpdate, nil
}

// DeleteUser deletes existing record by id
func (ops *operations) DeleteUser(id uint) (returnErr error) {

	var userCount int64
	if err := ops.db.Model(&models.User{}).Where("id = ?", id).Count(&userCount).Error; err != nil {
		ops.log.Errorf("Failed to determine  if an user with id %d is already registered: %v ", id, err)
		return appErrors.ErrInternal
	}
	if userCount == 0 {
		return appErrors.ErrUserDoesNotExist
	}

	userToDelete := models.User{DBModel: models.DBModel{ID: id}}
	if err := ops.db.Model(&models.User{}).Unscoped().Delete(&userToDelete).Error; err != nil {
		ops.log.Errorf("Failed to delete user with id %d: %v", id, err)
		return appErrors.ErrInternal
	}
	return nil
}

// FetchUsersWithPagination responds with users associated with currentPage of given size
func (ops *operations) FetchUsersWithPagination(currentPage int, pageSize int) (users []models.User,
	total int64,
	returnErr error) {

	offset := (currentPage - 1) * pageSize
	if err := ops.db.Model(&models.User{}).Count(&total).Error; err != nil {
		ops.log.Errorf("Failed to get the total count of users: %v", err)
		return nil, 0, appErrors.ErrInternal
	}
	if err := ops.db.Limit(pageSize).Offset(offset).Find(&users).Error; err != nil {
		ops.log.Errorf("Failed to fetch users: %v", err)
		return nil, 0, appErrors.ErrInternal
	}
	return
}

// FormatUserDetailsWithPageDetails...
func (ops *operations) FormatUserDetailsWithPageDetails(users []models.User,
	totalUsers int64, currentPage, pageSize int) models.PaginatedUserList {
	return models.PaginatedUserList{
		Data:        users,
		TotalItems:  totalUsers,
		CurrentPage: currentPage,
		PageSize:    pageSize,
	}
}

// ChangePassword sets passwordHash in DB for the user and resets temp_password flag
func (ops *operations) ChangePassword(email string, passwordHash string) (returnErr error) {
	var userCount int64
	if err := ops.db.Model(&models.User{}).Where("email = ?", email).Count(&userCount).Error; err != nil {
		ops.log.Errorf("Failed to determine if a user with email %s is already registered: %v ", email, err)
		return appErrors.ErrInternal
	}
	if userCount == 0 {
		return appErrors.ErrUserDoesNotExist
	}
	userToUpdate := map[string]interface{}{"email": email, "password_hash": passwordHash, "temp_password": false}
	if err := ops.db.Model(&models.User{}).Where("email = ?", email).Updates(userToUpdate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Handle any concurrent deletion as well
			return appErrors.ErrUserDoesNotExist
		}
		ops.log.Errorf("Failed to update password for the user with email %s: %v", email, err)
		return appErrors.ErrInternal
	}
	return nil
}
