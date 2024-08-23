package models

import (
	"time"
	"userservice/internal/utils"

	"gorm.io/gorm"
)

const (
	AttributeServiceName        = "name"
	AttributeServiceDescription = "description"
	AttributeServiceVersionTag  = "tag"
	AttributeServiceVersionInfo = "info"
)

var (
	ViewsScheduledSyncTime      = time.Duration(30 * time.Minute)
	ViewRefreshRequestCheckTime = time.Duration(2 * time.Second)
)

// Service represent service metadata with GORM field representation.
// VersionCount is precomputed and maintain, so we support reads from high scalable users.
type Service struct {
	DBModel
	Name         string `json:"name" gorm:"column:name;unique;not null" validate:"required"`
	Description  string `json:"description" gorm:"column:description"`
	VersionCount int    `json:"versionCount" gorm:"column:version_count"`
}

// TableName...
func (Service) TableName() string {
	return "service"
}

const (
	NameSortedServiceView string = "name_sorted_service"
)

// ServiceVersion represent version metadata with GORM field representation
type ServiceVersion struct {
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Service   Service        `json:"-" gorm:"foreignKey:ServiceID;references:ID"`
	ServiceID uint           `json:"-" gorm:"uniqueIndex:unique_composite;column:service_id"`
	Tag       string         `json:"tag" gorm:"uniqueIndex:unique_composite;column:tag;not null" validate:"required"`
	Info      string         `json:"info" gorm:"column:info"`
}

// TableName...
func (ServiceVersion) TableName() string {
	return "version"
}

// We consider global variables for payload templates, since
// there is not dependant variables getting initialized based on its value,
// we have only concurrent reads.
// even in worst case, we are ok with delayed point of initialization during package init.

// RegisterOrUpdateServicePayloadTemplate represents mandatory fields in service register/update request payload
var RegisterOrUpdateServicePayloadTemplate = utils.FieldTypeBinder{
	AttributeServiceName:        utils.String,
	AttributeServiceDescription: utils.String,
}

// RegisterVersionPayloadTemplate represents mandatory fields in service version register request payload
var RegisterVersionPayloadTemplate = utils.FieldTypeBinder{
	AttributeServiceVersionTag:  utils.String,
	AttributeServiceVersionInfo: utils.String,
}

// RegisterVersionPayloadTemplate represents mandatory fields in service version update request payload
var UpdateVersionPayloadTemplate = utils.FieldTypeBinder{
	AttributeServiceVersionInfo: utils.String,
}

// PaginatedServiceList...
type PaginatedServiceList struct {
	Data        []Service
	TotalItems  int64
	PageSize    int
	CurrentPage int
}

// PaginatedVersionList...
type PaginatedVersionList struct {
	Data        []ServiceVersion
	TotalItems  int64
	PageSize    int
	CurrentPage int
}

// ServiceOperations...
type ServiceOperations interface {
	SyncDataWithSortedViews()
	CheckIfServiceExist(uint) (bool, error)
	CheckIfVersionForServiceExist(uint, string) (bool, error)
	GetService(uint) (*Service, error)
	CreateService(string, string) (*Service, error)
	UpdateService(uint, string, string) (*Service, error)
	DeleteService(uint) error
	FetchServices(int, int, string, bool, bool) ([]Service, int64, error)
	FormatServiceDetailsWithPageDetails([]Service, int64, int, int) PaginatedServiceList
	GetServiceVersion(uint, string) (*ServiceVersion, error)
	CreateServiceVersion(uint, string, string) (*ServiceVersion, error)
	UpdateServiceVersion(uint, string, string) (*ServiceVersion, error)
	DeleteServiceVersion(uint, string) error
	FetchServiceVersionsInverted(uint, int, int) ([]ServiceVersion, int64, error)
	FormatVersionDetailsWithPageDetails([]ServiceVersion, int64, int, int) PaginatedVersionList
}
