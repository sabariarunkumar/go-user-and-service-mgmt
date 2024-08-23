package models

// UserRole kept simple as we are not doing UserRole management
type UserRole struct {
	Name        string `json:"name" gorm:"column:name;unique;not null"`
	Description string `json:"description" gorm:"column:description;unique;not null"`
}

// TableName...
func (UserRole) TableName() string {
	return "user_role"
}

// RoleOperations...
type RoleOperations interface {
	FetchRoles() ([]UserRole, error)
}
