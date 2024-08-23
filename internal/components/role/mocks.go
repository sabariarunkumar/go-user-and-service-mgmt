package role

import (
	"errors"
	"userservice/internal/models"
)

// RoleMockDefault...
type RoleMockDefault struct{}

// FetchRoles...
func (m *RoleMockDefault) FetchRoles() ([]models.UserRole, error) {
	return []models.UserRole{{Name: "basic", Description: "desc"}}, nil
}

// RoleMockForcedError...
type RoleMockForcedError struct {
	RoleMockDefault
}

// FetchRoles...
func (m *RoleMockForcedError) FetchRoles() ([]models.UserRole, error) {
	return nil, errors.New("connection error")
}
