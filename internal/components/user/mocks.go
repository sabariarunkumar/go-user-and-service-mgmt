package user

import (
	appErrors "userservice/internal/errors"
	"userservice/internal/models"

	"gorm.io/gorm"
)

// UserMock...
type UserMock struct {
	User                 *models.User
	SetInternalError     bool
	SetEmailOrIDNotFound bool
	SetDuplicateEmail    bool
	SetUserDoesntExist   bool
}

// GetUserByEmail...
func (m *UserMock) GetUserByEmail(string) (*models.User, error) {
	if m.SetInternalError {
		return nil, appErrors.ErrInternal
	} else if m.SetEmailOrIDNotFound {
		return nil, gorm.ErrRecordNotFound
	}
	return m.User, nil
}

// GetUser...
func (m *UserMock) GetUser(uint) (*models.User, error) {
	if m.SetInternalError {
		return nil, appErrors.ErrInternal
	} else if m.SetEmailOrIDNotFound {
		return nil, gorm.ErrRecordNotFound
	}
	return m.User, nil
}

// CreateUser...
func (m *UserMock) CreateUser(string, string, string, string) error {
	if m.SetInternalError {
		return appErrors.ErrInternal
	} else if m.SetDuplicateEmail {
		return appErrors.ErrUserWithSameEmailAlreadyExists
	}
	return nil
}

// UpdateUser
func (m *UserMock) UpdateUser(uint, string, string, string) (*models.User, error) {
	if m.SetInternalError {
		return nil, appErrors.ErrInternal
	} else if m.SetDuplicateEmail {
		return nil, appErrors.ErrUserWithSameEmailAlreadyExists
	} else if m.SetUserDoesntExist {
		return nil, appErrors.ErrUserDoesNotExist
	}
	return m.User, nil
}

// DeleteUser
func (m *UserMock) DeleteUser(uint) error {
	if m.SetInternalError {
		return appErrors.ErrInternal
	} else if m.SetUserDoesntExist {
		return appErrors.ErrUserDoesNotExist
	}
	return nil
}

// FetchUsersWithPagination
func (m *UserMock) FetchUsersWithPagination(int, int) ([]models.User, int64, error) {
	if m.SetInternalError {
		return nil, 0, appErrors.ErrInternal
	}
	var users []models.User
	users = append(users, *m.User)
	return users, 1, nil
}

// FormatUserDetailsWithPageDetails
func (m *UserMock) FormatUserDetailsWithPageDetails(users []models.User, total int64, page int, pageSize int) models.PaginatedUserList {
	return models.PaginatedUserList{
		Data:        users,
		TotalItems:  total,
		CurrentPage: page,
		PageSize:    pageSize,
	}
}

// ChangePassword
func (m *UserMock) ChangePassword(string, string) error {
	if m.SetInternalError {
		return appErrors.ErrInternal
	} else if m.SetUserDoesntExist {
		return appErrors.ErrUserDoesNotExist
	}
	return nil
}
