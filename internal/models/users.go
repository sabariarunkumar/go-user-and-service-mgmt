package models

import "userservice/internal/utils"

const (
	AttributeName     = "name"
	AttributeEmail    = "email"
	AttributePassword = "password"
	AttributeRole     = "role"
)

// User represent user metadata with GORM field representation
type User struct {
	DBModel
	Name                string `json:"name" gorm:"column:name"`
	Email               string `json:"email" gorm:"column:email;unique;not null"`
	Role                string `json:"role" gorm:"column:role;not null"`
	PasswordHash        string `json:"-" gorm:"column:password_hash"`
	IsTemporaryPassword bool   `json:"-" gorm:"type:boolean;column:temp_password"`
}

// TableName...
func (User) TableName() string {
	return "user"
}

// We consider global variables for payload templates, since
// there is not dependant variables getting initialized based on its value,
// we have only concurrent reads.
// even in worst case, we are ok with delayed point of initialization during package init.

// LoginPayloadTemplate represents mandatory fields in user request payload
var LoginPayloadTemplate = utils.FieldTypeBinder{
	AttributeEmail:    utils.String,
	AttributePassword: utils.String,
}

// RegisterOrUpdateUserPayloadTemplate represents mandatory fields in user registration/update payload
var RegisterOrUpdateUserPayloadTemplate = utils.FieldTypeBinder{
	AttributeName:  utils.String,
	AttributeEmail: utils.String,
	AttributeRole:  utils.String,
}

// LoginPayloadTemplate represents mandatory fields in password change payload
var PasswordChangePayloadTemplate = utils.FieldTypeBinder{
	AttributePassword: utils.String,
}

// PaginatedUserList...
type PaginatedUserList struct {
	Data        []User
	TotalItems  int64
	PageSize    int
	CurrentPage int
}

// UserOperations...
type UserOperations interface {
	GetUserByEmail(string) (*User, error)
	GetUser(uint) (*User, error)
	CreateUser(string, string, string, string) error
	UpdateUser(uint, string, string, string) (*User, error)
	DeleteUser(uint) error
	FetchUsersWithPagination(int, int) ([]User, int64, error)
	FormatUserDetailsWithPageDetails([]User, int64, int, int) PaginatedUserList
	ChangePassword(string, string) error
}
