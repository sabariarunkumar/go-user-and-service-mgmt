package service

import (
	"context"
	"errors"
	"log"
	"math"
	"strings"
	"sync"
	"time"
	appErrors "userservice/internal/errors"
	"userservice/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// operations...
type operations struct {
	db             *gorm.DB
	log            *zap.SugaredLogger
	ctx            context.Context
	mux            sync.RWMutex
	toRefreshViews bool
}

// newOperations initializes service operation handler along with a sync goroutine
func newOperations(ctx context.Context, wg *sync.WaitGroup, db *gorm.DB, log *zap.SugaredLogger) *operations {
	ops := operations{db: db, log: log, ctx: ctx}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ops.SyncDataWithSortedViews()
	}()
	return &ops
}

// reverseServiceSlice...
var reverseServiceSlice = func(slice []models.Service) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// reverseServiceVersionSlice...
var reverseServiceVersionSlice = func(slice []models.ServiceVersion) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// SyncDataWithSortedViews watches for any change in Service table
// If any change, it will refresh its name sorted materialized view of ServiceVersions table.
// We could do these as a part of individual transaction involving changes to Service table,
// here we are making it modular and handle it asynchronously because if in future multiple column
// sorting is involved or multiple queries are encountered in system, we would end up in issue-ing
// multiple refresh requests at once. Hence we ensure only one request at max per scheduledSyncTicker ticker.
func (ops *operations) SyncDataWithSortedViews() {

	refreshViews := func() (err error) {
		if err = ops.db.Exec("REFRESH MATERIALIZED VIEW " + models.NameSortedServiceView).Error; err != nil {
			ops.log.Errorf("could not refresh materialized view %s: %v", models.NameSortedServiceView, err)
		}
		return
	}

	refreshViewsWithAttempts := func() (err error) {
		for attempts := 0; attempts < 3; attempts++ {
			err = refreshViews()
			if err == nil {
				return nil
			}
			time.Sleep(time.Duration(math.Pow(2, float64(attempts)) * float64(time.Second)))
		}
		return err
	}

	if err := refreshViewsWithAttempts(); err != nil {
		log.Fatalf("Initial check to identify if Service table and its views are in sync failed %v", err)
	}

	// Using scheduled syncs, for eventual consistency, in case of system failure
	// when Services table is updated right after there is a sudden crash
	scheduledSyncTicker := time.NewTicker(models.ViewsScheduledSyncTime)
	defer scheduledSyncTicker.Stop()

	viewRefreshRequestCheckTicker := time.NewTicker(models.ViewRefreshRequestCheckTime)
	defer viewRefreshRequestCheckTicker.Stop()
	for {
		select {
		case <-ops.ctx.Done():
			return

		case <-scheduledSyncTicker.C:
			if err := refreshViewsWithAttempts(); err != nil {
				log.Fatalf("failed to sync view table %v", err)
			}
		case <-viewRefreshRequestCheckTicker.C:
			var toRefresh bool
			ops.mux.RLock()
			toRefresh = ops.toRefreshViews
			ops.mux.RUnlock()
			if toRefresh {
				err := refreshViews()
				if err == nil {
					ops.mux.Lock()
					ops.toRefreshViews = false
					ops.mux.Unlock()
				}
				// In case of error, retry will happen at next viewRefreshRequestCheckTicker time
			}
		}
	}
}

// CheckIfServiceExist checks if service of given ID exists
func (ops *operations) CheckIfServiceExist(id uint) (exists bool, returnErr error) {
	var serviceCount int64
	if err := ops.db.Model(&models.Service{}).Where("id = ?", id).Count(&serviceCount).Error; err != nil {
		ops.log.Errorf("Failed to determine if a service with ID %d is already registered: %v", id, err)
		returnErr = appErrors.ErrInternal
	}
	if serviceCount != 0 {
		exists = true
	}
	return
}

// CheckIfVersionForServiceExist checks if versionTag for given service ID exist
func (ops *operations) CheckIfVersionForServiceExist(serviceID uint, versionTag string) (exists bool, returnErr error) {
	var serviceVersionCount int64
	if err := ops.db.Model(&models.ServiceVersion{}).Where("tag = ? and service_id = ?",
		versionTag, serviceID).Count(&serviceVersionCount).Error; err != nil {
		ops.log.Errorf("Failed to determine if version (Tag: %s) registered for service (ID: %d) : %v",
			versionTag, serviceID, err)
		returnErr = appErrors.ErrInternal
	}
	if serviceVersionCount != 0 {
		exists = true
	}
	return
}

// GetService fetch service of given ID
func (ops *operations) GetService(id uint) (service *models.Service, returnErr error) {
	service = new(models.Service)
	gormErr := ops.db.Where("id = ?", id).First(service).Error
	if gormErr != nil {
		service = nil
		if errors.Is(gormErr, gorm.ErrRecordNotFound) {
			returnErr = appErrors.ErrServiceDoesNotExist
		} else {
			ops.log.Errorf("Failed to fetch service record by id %s : %v ", id, gormErr)
			returnErr = appErrors.ErrInternal
		}
	}
	return
}

// CreateService creates service record in DB  with necessary metadata.
// Since service creation happens seldom, we have additional DB call
// to check if record exist with same service name rather than waiting for DB to report uniqueKey constrain.
// We still need to handle duplicate record constrain gracefully if create request happens at once
// Materialized View refresh will be scheduled accordingly.
func (ops *operations) CreateService(name string, description string) (*models.Service, error) {

	var userWithSameServiceName int64 = 0
	if gormErr := ops.db.Model(&models.Service{}).Where("name = ?", name).
		Count(&userWithSameServiceName).Error; gormErr != nil {
		ops.log.Errorf("Failed to determine if service %s exists: %v", name, gormErr)
		return nil, appErrors.ErrInternal
	}
	if userWithSameServiceName == 1 {
		return nil, appErrors.ErrServiceAlreadyExists
	}

	newService := &models.Service{Name: name, Description: description}
	if gormErr := ops.db.Model(&models.Service{}).Create(newService).Error; gormErr != nil {
		if strings.Contains(gormErr.Error(), appErrors.ErrUniqueKeyConstrainViolation.Error()) {
			return nil, appErrors.ErrServiceAlreadyExists
		}
		ops.log.Errorf("Failed to create service %s: %v", name, gormErr)
		return nil, appErrors.ErrInternal
	}

	ops.mux.Lock()
	ops.toRefreshViews = true
	ops.mux.Unlock()
	return newService, nil
}

// UpdateService updates existing service record in DB with necessary metadata.
// Since service update happens seldom, we have additional DB call
// to check if record with requested new service version tag exist rather than
// waiting for DB to report uniqueKey constrain.
// We still need to handle duplicate record constrain gracefully in distributed/concurrent environment.
func (ops *operations) UpdateService(id uint, name string, description string) (
	*models.Service,
	error) {

	var exists bool
	exists, returnErr := ops.CheckIfServiceExist(id)
	if returnErr != nil {
		return nil, returnErr
	}
	if !exists {
		return nil, appErrors.ErrServiceDoesNotExist
	}

	var userWithSameServiceName int64 = 0
	if gormErr := ops.db.Model(&models.Service{}).Where("name = ? and id != ?", name, id).
		Count(&userWithSameServiceName).Error; gormErr != nil {
		ops.log.Errorf("Failed to update service %s: %v", name, gormErr)
		return nil, appErrors.ErrInternal
	}
	if userWithSameServiceName == 1 {
		return nil, appErrors.ErrServiceAlreadyExists
	}

	serviceToUpdate := &models.Service{Name: name, Description: description, DBModel: models.DBModel{ID: id}}
	if gormErr := ops.db.Model(&models.Service{}).Where("id = ?", id).
		Updates(serviceToUpdate).Error; gormErr != nil {
		if strings.Contains(gormErr.Error(), appErrors.ErrUniqueKeyConstrainViolation.Error()) {
			return nil, appErrors.ErrServiceAlreadyExists
		} else if errors.Is(gormErr, gorm.ErrRecordNotFound) {
			// Handle any concurrent deletion
			return nil, appErrors.ErrServiceDoesNotExist
		}
		ops.log.Errorf("Failed to update service %s: %v", name, gormErr)
		return nil, appErrors.ErrInternal
	}

	ops.mux.Lock()
	ops.toRefreshViews = true
	ops.mux.Unlock()
	return serviceToUpdate, nil
}

// DeleteUser deletes existing service record by id and associated version records
func (ops *operations) DeleteService(id uint) error {
	var exists bool
	exists, returnErr := ops.CheckIfServiceExist(id)
	if returnErr != nil {
		return returnErr
	}
	if !exists {
		return appErrors.ErrServiceDoesNotExist
	}

	returnErr = ops.db.Transaction(func(tx *gorm.DB) error {
		gormErr := tx.Where("service_id = ?", id).Unscoped().Delete(&models.ServiceVersion{}).Error
		if gormErr != nil {
			if !errors.Is(gormErr, gorm.ErrRecordNotFound) {
				ops.log.Errorf("Failed to delete versions associated with service [ID:%d]: %v", id, gormErr)
				return appErrors.ErrInternal
			}
		}
		if err := tx.Unscoped().Delete(&models.Service{}, id).Error; err != nil {
			ops.log.Errorf("Failed to delete service [ID:%d]: %v", id, err)
			return appErrors.ErrInternal
		}
		return nil
	})

	if returnErr == nil {
		ops.mux.Lock()
		ops.toRefreshViews = true
		ops.mux.Unlock()
	}
	return returnErr
}

// FetchServices responds with services associated with currentPage of given size and sorting order.
// InvertedFetch and string searches are supported
// Non existing pages are returning with empty service list, rather than nil, and expected caller to handle it
func (ops *operations) FetchServices(
	currentPage int,
	pageSize int,
	searchString string,
	invertedFetch bool,
	fetchNameSortedServices bool) (services []models.Service, total int64, returnErr error) {

	var (
		offset           int
		limit            = pageSize
		referenceDBTable = models.Service{}.TableName()
	)

	if fetchNameSortedServices {
		referenceDBTable = models.NameSortedServiceView
	}

	if err := ops.db.Table(referenceDBTable).Where("name like ?", searchString).Count(&total).Error; err != nil {
		ops.log.Errorf("Failed to get the total count of services: %v", err)
		return nil, 0, appErrors.ErrInternal
	}

	if invertedFetch {
		offset = int(total) - (currentPage * pageSize)
		if offset < 0 {
			limit = int(total) % pageSize
			if offset >= limit-pageSize {
				offset = 0
			} else {
				// handle non-existing page requests
				services = make([]models.Service, 0)
				return
			}
		}
	} else {
		offset = (currentPage - 1) * pageSize
	}

	if err := ops.db.Table(referenceDBTable).Where("name like ?", searchString).
		Limit(limit).Offset(offset).Find(&services).Error; err != nil {
		ops.log.Errorf("Failed to fetch services: %v", err)
		return nil, 0, appErrors.ErrInternal
	}

	if len(services) != 0 && invertedFetch {
		reverseServiceSlice(services)
	}
	return
}

// FormatServiceDetailsWithPageDetails...
func (ops *operations) FormatServiceDetailsWithPageDetails(
	services []models.Service,
	totalServices int64,
	currentPage,
	pageSize int) models.PaginatedServiceList {
	return models.PaginatedServiceList{
		Data:        services,
		TotalItems:  totalServices,
		CurrentPage: currentPage,
		PageSize:    pageSize,
	}
}

// GetService fetch service of given ID and version tag
func (ops *operations) GetServiceVersion(serviceID uint, versionTag string) (
	version *models.ServiceVersion, returnErr error) {
	version = new(models.ServiceVersion)
	gormErr := ops.db.Where("tag = ? and service_id = ?", versionTag, serviceID).First(version).Error
	if gormErr != nil {
		version = nil
		if errors.Is(gormErr, gorm.ErrRecordNotFound) {
			returnErr = appErrors.ErrServiceVersionDoesNotExist
		} else {
			ops.log.Errorf("Failed to fetch version record with tag %s and service_id %d: %v",
				versionTag, serviceID, gormErr)
			returnErr = appErrors.ErrInternal
		}
	}
	return
}

// CreateServiceVersion creates version record in DB  with necessary metadata.
// Since service creation happens seldom, we have additional DB call
// to check if record exist with same version tag rather than waiting for DB to report uniqueKey constrain.
// We still need to handle duplicate record constrain gracefully if create request happens at once
func (ops *operations) CreateServiceVersion(serviceID uint,
	versionTag string,
	info string) (*models.ServiceVersion, error) {

	var serviceWithSameVersion int64 = 0
	if gormErr := ops.db.Model(&models.ServiceVersion{}).Where("tag = ? and service_id = ?", versionTag, serviceID).
		Count(&serviceWithSameVersion).Error; gormErr != nil {
		ops.log.Errorf("Failed to determine if service version tag %s exists for service [ID:%d]: %v",
			versionTag, serviceID, gormErr)
		return nil, appErrors.ErrInternal
	}
	if serviceWithSameVersion == 1 {
		return nil, appErrors.ErrServiceVersionAlreadyExists
	}
	newVersion := &models.ServiceVersion{Tag: versionTag, Info: info, ServiceID: serviceID}
	returnErr := ops.db.Transaction(func(tx *gorm.DB) error {
		if gormErr := tx.Create(newVersion).Error; gormErr != nil {
			if strings.Contains(gormErr.Error(), appErrors.ErrUniqueKeyConstrainViolation.Error()) {
				return appErrors.ErrServiceVersionAlreadyExists
			}
			ops.log.Errorf("Failed to create version %s for service[ID:%d] : %v", versionTag, serviceID, gormErr)
			return appErrors.ErrInternal
		}
		if err := tx.Model(models.Service{}).Where("id = ?",
			serviceID).UpdateColumn("version_count", gorm.Expr("version_count + ?", 1)).Error; err != nil {
			return appErrors.ErrInternal
		}
		return nil
	})
	if returnErr == nil {
		ops.mux.Lock()
		ops.toRefreshViews = true
		ops.mux.Unlock()
		return newVersion, nil
	}

	return nil, returnErr
}

// UpdateServiceVersion updated version record in DB  with necessary metadata.
// Since service creation happens seldom, we have additional DB call
// to check if record with requested new service name exist with same version tag rather
// than waiting for DB to report uniqueKey constrain.
// We still need to handle duplicate record constrain gracefully if create request happens at once
func (ops *operations) UpdateServiceVersion(
	serviceID uint,
	versionTag string,
	info string) (*models.ServiceVersion, error) {
	exist, returnErr := ops.CheckIfVersionForServiceExist(serviceID, versionTag)
	if returnErr != nil {
		return nil, returnErr
	}
	if !exist {
		return nil, appErrors.ErrServiceVersionDoesNotExist
	}
	serviceToUpdate := &models.ServiceVersion{Tag: versionTag, ServiceID: serviceID, Info: info}
	if gormErr := ops.db.Where("tag = ? and service_id = ?", versionTag, serviceID).
		Updates(serviceToUpdate).Error; gormErr != nil {
		if errors.Is(gormErr, gorm.ErrRecordNotFound) {
			return nil, appErrors.ErrServiceVersionDoesNotExist
		}
		//  consider foreign key constraint violation if say corresponding service record is deleted
		ops.log.Errorf("Failed to create version %s for service ID : %v", versionTag, gormErr)
		return nil, appErrors.ErrInternal
	}
	return serviceToUpdate, nil

}

// DeleteUser deletes existing service version record by id and decrement version records count in Service Table
func (ops *operations) DeleteServiceVersion(serviceID uint, versionTag string) error {
	exist, returnErr := ops.CheckIfVersionForServiceExist(serviceID, versionTag)
	if returnErr != nil {
		return returnErr
	}
	if !exist {
		return appErrors.ErrServiceVersionDoesNotExist
	}

	returnErr = ops.db.Transaction(func(tx *gorm.DB) error {
		gormErr := tx.Unscoped().Where("tag = ? and service_id = ?", versionTag, serviceID).
			Delete(&models.ServiceVersion{}).Error
		if gormErr != nil {
			if errors.Is(gormErr, gorm.ErrRecordNotFound) {
				return appErrors.ErrServiceVersionDoesNotExist
			}
			ops.log.Errorf("Failed to delete version %s associated with service ID %d: %v",
				versionTag, serviceID, gormErr)
			return appErrors.ErrInternal
		}
		if err := tx.Model(models.Service{}).Where("id = ?", serviceID).
			UpdateColumn("version_count", gorm.Expr("version_count - ?", 1)).Error; err != nil {
			return appErrors.ErrInternal
		}
		return nil
	})

	if returnErr == nil {
		ops.mux.Lock()
		ops.toRefreshViews = true
		ops.mux.Unlock()
	}
	return returnErr
}

// FetchServiceVersionsInverted responds with services associated with currentPage of given size [inverted]
func (ops *operations) FetchServiceVersionsInverted(
	id uint,
	currentPage int,
	pageSize int) (serviceVersions []models.ServiceVersion, total int64, returnErr error) {

	limit := pageSize
	if err := ops.db.Model(&models.ServiceVersion{}).Where("service_id = ?", id).Count(&total).Error; err != nil {
		ops.log.Errorf("Failed to get the total count of versions for service %d: %v", id, err)
		return nil, 0, appErrors.ErrInternal
	}

	offset := int(total) - (currentPage * pageSize)
	if offset < 0 {
		limit = int(total) % pageSize
		if offset >= limit-pageSize {
			offset = 0
		} else {
			serviceVersions = make([]models.ServiceVersion, 0)
			return
		}
	}

	if err := ops.db.Where("service_id = ?", id).
		Limit(limit).Offset(offset).Find(&serviceVersions).Error; err != nil {
		ops.log.Errorf("Failed to fetch service versions for service %d: %v", id, err)
		return nil, 0, appErrors.ErrInternal
	}
	if len(serviceVersions) != 0 {
		reverseServiceVersionSlice(serviceVersions)
	}
	return
}

// FormatVersionDetailsWithPageDetails...
func (ops *operations) FormatVersionDetailsWithPageDetails(serviceVersions []models.ServiceVersion,
	totalServices int64, currentPage, pageSize int) models.PaginatedVersionList {
	return models.PaginatedVersionList{
		Data:        serviceVersions,
		TotalItems:  totalServices,
		CurrentPage: currentPage,
		PageSize:    pageSize,
	}
}
