package service

import (
	appErrors "userservice/internal/errors"
	"userservice/internal/models"
)

// MockFuncs...
type MockFuncs map[string]struct{}

const (
	ServiceExistenceFn        = "CheckIfServiceExist"
	ServiceVersionExistenceFn = "CheckIfVersionForServiceExist"
	GetServiceFn              = "GetService"
	CreateServiceFn           = "CreateService"
	UpdateServiceFn           = "UpdateService"
	DeleteServiceFn           = "DeleteService"
	FetchServiceFn            = "FetchServices"
	GetServiceVersionFn       = "GetServiceVersion"
	CreateServiceVersionFn    = "CreateServiceVersion"
	UpdateServiceVersionFn    = "UpdateServiceVersion"
	DeleteServiceVersionFn    = "DeleteServiceVersion"
	FetchServiceVersionFn     = "FetchServiceVersionsInverted"
)

// ServiceAndVersionMock...
type ServiceAndVersionMock struct {
	Service               *models.Service
	Version               *models.ServiceVersion
	SetInternalError      MockFuncs
	SetRecordNotFound     MockFuncs
	SetRecordAlreadyExist MockFuncs
}

// SyncDataWithSortedViews...
func (m *ServiceAndVersionMock) SyncDataWithSortedViews() {

}

// CheckIfServiceExist...
func (m *ServiceAndVersionMock) CheckIfServiceExist(uint) (bool, error) {
	if _, ok := m.SetInternalError[ServiceExistenceFn]; ok {
		return false, appErrors.ErrInternal
	} else if _, ok := m.SetRecordNotFound[ServiceExistenceFn]; ok {
		return false, nil
	}
	return true, nil
}

// CheckIfVersionForServiceExist...
func (m *ServiceAndVersionMock) CheckIfVersionForServiceExist(uint, string) (bool, error) {
	return true, nil
}

// GetService...
func (m *ServiceAndVersionMock) GetService(uint) (*models.Service, error) {
	if _, ok := m.SetInternalError[GetServiceFn]; ok {
		return nil, appErrors.ErrInternal
	} else if _, ok := m.SetRecordNotFound[GetServiceFn]; ok {
		return nil, appErrors.ErrServiceDoesNotExist
	}
	return m.Service, nil
}

// CreateService...
func (m *ServiceAndVersionMock) CreateService(string, string) (*models.Service, error) {
	if _, ok := m.SetInternalError[CreateServiceFn]; ok {
		return nil, appErrors.ErrInternal
	} else if _, ok := m.SetRecordAlreadyExist[CreateServiceFn]; ok {
		return nil, appErrors.ErrServiceAlreadyExists
	}
	return m.Service, nil
}

// UpdateService...
func (m *ServiceAndVersionMock) UpdateService(uint, string, string) (*models.Service, error) {
	if _, ok := m.SetInternalError[UpdateServiceFn]; ok {
		return nil, appErrors.ErrInternal
	} else if _, ok := m.SetRecordNotFound[UpdateServiceFn]; ok {
		return nil, appErrors.ErrServiceDoesNotExist
	} else if _, ok := m.SetRecordAlreadyExist[UpdateServiceFn]; ok {
		return nil, appErrors.ErrServiceAlreadyExists
	}
	return m.Service, nil
}

// DeleteService...
func (m *ServiceAndVersionMock) DeleteService(uint) error {
	if _, ok := m.SetInternalError[DeleteServiceFn]; ok {
		return appErrors.ErrInternal
	} else if _, ok := m.SetRecordNotFound[DeleteServiceFn]; ok {
		return appErrors.ErrServiceDoesNotExist
	}
	return nil
}

// FetchServices...
func (m *ServiceAndVersionMock) FetchServices(int, int, string, bool, bool) ([]models.Service, int64, error) {
	if _, ok := m.SetInternalError[FetchServiceFn]; ok {
		return nil, 0, appErrors.ErrInternal
	}
	var services []models.Service
	services = append(services, *m.Service)
	return services, 1, nil
}

// FormatServiceDetailsWithPageDetails...
func (m *ServiceAndVersionMock) FormatServiceDetailsWithPageDetails(services []models.Service, total int64, page int, pageSize int) models.PaginatedServiceList {
	return models.PaginatedServiceList{
		Data:        services,
		TotalItems:  total,
		CurrentPage: page,
		PageSize:    pageSize,
	}
}

// GetServiceVersion...
func (m *ServiceAndVersionMock) GetServiceVersion(uint, string) (*models.ServiceVersion, error) {
	if _, ok := m.SetInternalError[GetServiceVersionFn]; ok {
		return nil, appErrors.ErrInternal
	} else if _, ok := m.SetRecordNotFound[GetServiceVersionFn]; ok {
		return nil, appErrors.ErrServiceVersionDoesNotExist
	}
	return m.Version, nil
}

// CreateServiceVersion...
func (m *ServiceAndVersionMock) CreateServiceVersion(uint, string, string) (*models.ServiceVersion, error) {
	if _, ok := m.SetInternalError[CreateServiceVersionFn]; ok {
		return nil, appErrors.ErrInternal
	} else if _, ok := m.SetRecordAlreadyExist[CreateServiceVersionFn]; ok {
		return nil, appErrors.ErrServiceVersionAlreadyExists
	}
	return m.Version, nil
}

// UpdateServiceVersion...
func (m *ServiceAndVersionMock) UpdateServiceVersion(uint, string, string) (*models.ServiceVersion, error) {
	if _, ok := m.SetInternalError[UpdateServiceVersionFn]; ok {
		return nil, appErrors.ErrInternal
	} else if _, ok := m.SetRecordNotFound[UpdateServiceVersionFn]; ok {
		return nil, appErrors.ErrServiceVersionDoesNotExist
	}
	return m.Version, nil
}

// DeleteServiceVersion...
func (m *ServiceAndVersionMock) DeleteServiceVersion(uint, string) error {
	if _, ok := m.SetInternalError[DeleteServiceVersionFn]; ok {
		return appErrors.ErrInternal
	} else if _, ok := m.SetRecordNotFound[DeleteServiceVersionFn]; ok {
		return appErrors.ErrServiceVersionDoesNotExist
	}
	return nil
}

// FetchServiceVersionsInverted...
func (m *ServiceAndVersionMock) FetchServiceVersionsInverted(uint, int, int) ([]models.ServiceVersion, int64, error) {
	if _, ok := m.SetInternalError[FetchServiceVersionFn]; ok {
		return nil, 0, appErrors.ErrInternal
	}
	var versions []models.ServiceVersion
	versions = append(versions, *m.Version)
	return versions, 1, nil
}

// FormatVersionDetailsWithPageDetails...
func (m *ServiceAndVersionMock) FormatVersionDetailsWithPageDetails(versions []models.ServiceVersion, total int64, page int, pageSize int) models.PaginatedVersionList {
	return models.PaginatedVersionList{
		Data:        versions,
		TotalItems:  total,
		CurrentPage: page,
		PageSize:    pageSize,
	}
}
