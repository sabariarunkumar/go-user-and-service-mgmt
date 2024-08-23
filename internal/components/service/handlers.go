package service

import (
	"fmt"
	"strconv"
	appErrors "userservice/internal/errors"
	"userservice/internal/models"
	"userservice/internal/utils"

	"net/http"

	"github.com/gin-gonic/gin"
)

// getServiceByID fetches service record in DB for the given id
func (h *Handler) getServiceByID(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var serviceId uint
	if _, err := fmt.Sscanf(id, "%d", &serviceId); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service Fetch ID should be numerical"))
		return
	}
	service, err := h.operations.GetService(serviceId)
	if err != nil {
		if err == appErrors.ErrInternal {
			c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
			return
		}
		c.JSON(http.StatusNotFound, utils.FormatErrorResponse(fmt.Sprintf("Service[ID:%d] doesn't exist", serviceId)))
		return
	}
	c.JSON(http.StatusOK, service)
}

// addService configures new service
// Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) addService(c *gin.Context) {
	var serviceToAdd map[string]interface{}
	if err := c.BindJSON(&serviceToAdd); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service creation payload is invalid; Expected JSON payload"))
		return
	}

	if !utils.EnsureFieldsStrictlyExists(serviceToAdd, models.RegisterOrUpdateServicePayloadTemplate) {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse(fmt.Sprintf("Service creation payload is invalid; Strictly Allowed Params: %v",
				utils.ConvertFieldTypeToString(models.RegisterOrUpdateServicePayloadTemplate))))
		return
	}

	if serviceToAdd[models.AttributeServiceName] == "" {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse(fmt.Sprintf("Service Metadata Creation payload is invalid: %v",
				appErrors.ErrServiceNameEmpty)))
		return
	}

	createdService, err := h.operations.CreateService(serviceToAdd[models.AttributeServiceName].(string),
		serviceToAdd[models.AttributeServiceDescription].(string))
	if err != nil {
		if err == appErrors.ErrServiceAlreadyExists {
			c.JSON(http.StatusConflict, utils.FormatErrorResponse(
				fmt.Sprintf("Service %s already exists", serviceToAdd[models.AttributeServiceName].(string))))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(
			appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	c.JSON(http.StatusCreated, createdService)
}

// updateService update service in DB.
//
//	Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) updateService(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var serviceID uint
	if _, err := fmt.Sscanf(id, "%d", &serviceID); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service ID should be numerical"))
		return
	}

	var serviceToUpdate map[string]interface{}
	if err := c.BindJSON(&serviceToUpdate); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service Metadata update payload is invalid; Expected JSON payload"))
		return
	}

	if !utils.EnsureFieldsStrictlyExists(serviceToUpdate, models.RegisterOrUpdateServicePayloadTemplate) {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse(fmt.Sprintf("Service Metadata update payload is invalid; Strictly Allowed Params: %v",
				utils.ConvertFieldTypeToString(models.RegisterOrUpdateServicePayloadTemplate))))
		return
	}

	if serviceToUpdate[models.AttributeServiceName] == "" {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse(fmt.Sprintf("Service Metadata update payload is invalid: %v",
				appErrors.ErrServiceNameEmpty)))
		return
	}

	updateService, err := h.operations.UpdateService(serviceID,
		serviceToUpdate[models.AttributeServiceName].(string), serviceToUpdate[models.AttributeServiceDescription].(string))
	if err != nil {
		if err == appErrors.ErrServiceDoesNotExist {
			c.JSON(http.StatusNotFound, utils.FormatErrorResponse(fmt.Sprintf("Service %s doesn't exists",
				serviceToUpdate[models.AttributeServiceName].(string))))
			return
		} else if err == appErrors.ErrServiceAlreadyExists {
			c.JSON(http.StatusConflict, utils.FormatErrorResponse(fmt.Sprintf("Service %s already exists",
				serviceToUpdate[models.AttributeServiceName].(string))))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	// UI can reflect ServiceHub page to get the updated services list
	c.JSON(http.StatusOK, updateService)
}

// deleteService deletes service from system
// Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) deleteService(c *gin.Context) {
	id := c.Param(models.QueryParamID)

	var serviceID uint
	if _, err := fmt.Sscanf(id, "%d", &serviceID); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service ID should be numerical"))
		return
	}

	err := h.operations.DeleteService(serviceID)
	if err != nil {
		if err == appErrors.ErrServiceDoesNotExist {
			c.JSON(http.StatusNotFound, utils.FormatGenericResponse(fmt.Sprintf("Service[ID:%d] doesn't exist", serviceID)))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}

	// UI can reflect ServiceHub page to get the updated services list
	c.JSON(http.StatusOK, utils.FormatGenericResponse("Service deleted from system"))
}

// fetchServices list the services in system
func (h *Handler) fetchServices(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "0")
	page, paramErr := strconv.Atoi(pageStr)
	if paramErr != nil || page < 0 {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Request Path contains invalid page number, choose positive numerical value"))
		return
	}
	// page 0 or unset page parameter will represent the first page
	if page == 0 {
		page = 1
	}

	pageSizeStr := c.DefaultQuery("size", models.DefaultPageSize)
	pageSize, paramErr := strconv.Atoi(pageSizeStr)
	if paramErr != nil || pageSize < 0 {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Request Path contains invalid page size, choose positive numerical value"))
		return
	}

	var (
		sortBy                  string
		users                   []models.Service
		totalEntries            int64
		fetchErr                error
		invertedFetch           bool
		fetchNameSortedServices bool
	)
	searchStr := fmt.Sprintf("%%%s%%", c.DefaultQuery(models.AttributeServiceName, ""))

	sortBy = c.DefaultQuery("sort_by", "")
	if sortBy != "" && sortBy != models.AttributeServiceName && sortBy != "date" {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Request Path contains invalid sort_by value, choose name or date"))
		return
	}
	// default search is based on the date when the service is configured in system
	if sortBy == models.AttributeServiceName {
		fetchNameSortedServices = true
	}

	getInverted := c.DefaultQuery("inverted", "")
	if getInverted != "" && getInverted != "true" && getInverted != "false" {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Request Path contains invalid inverted value, choose true or false"))
		return
	}
	if getInverted == "true" {
		invertedFetch = true
	}

	users, totalEntries, fetchErr = h.operations.FetchServices(
		page, pageSize, searchStr, invertedFetch, fetchNameSortedServices)
	if fetchErr != nil {
		c.JSON(http.StatusInternalServerError,
			utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	result := h.operations.FormatServiceDetailsWithPageDetails(users, totalEntries, page, pageSize)
	c.JSON(http.StatusOK, result)
}

// getServiceVersion fetches service version record in DB for the given id
func (h *Handler) getServiceVersion(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var serviceID uint
	if _, err := fmt.Sscanf(id, "%d", &serviceID); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service ID should be numerical"))
		return
	}

	tag := c.Param(models.AttributeServiceVersionTag)
	if len(tag) == 0 {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Expected Non-empty Version Tag as a part of URL"))
		return
	}

	exists, err := h.operations.CheckIfServiceExist(serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound,
			utils.FormatErrorResponse(fmt.Sprintf("service[ID:%d] doesn't exist", serviceID)))
		return
	}

	version, err := h.operations.GetServiceVersion(serviceID, tag)
	if err != nil {
		if err == appErrors.ErrServiceVersionDoesNotExist {
			c.JSON(http.StatusNotFound,
				utils.FormatErrorResponse(
					fmt.Sprintf("version tag %s doesn't exist for service ID %d", tag, serviceID)))
		} else {
			c.JSON(http.StatusInternalServerError,
				utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, version)
}

// addServiceVersion configures new service version
// Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) addServiceVersion(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var serviceID uint
	if _, err := fmt.Sscanf(id, "%d", &serviceID); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service ID should be numerical"))
		return
	}
	var serviceVersionToAdd map[string]interface{}
	if err := c.BindJSON(&serviceVersionToAdd); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service Version creation payload is invalid; Expected JSON payload"))
		return
	}

	if !utils.EnsureFieldsStrictlyExists(serviceVersionToAdd, models.RegisterVersionPayloadTemplate) {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse(
				fmt.Sprintf("Service Version creation payload is invalid; Strictly Allowed Params: %v",
					utils.ConvertFieldTypeToString(models.RegisterVersionPayloadTemplate))))
		return
	}
	if serviceVersionToAdd[models.AttributeServiceVersionTag] == "" {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse(
				fmt.Sprintf("Service Version creation payload is invalid: %v",
					appErrors.ErrVersionTagEmpty)))
		return
	}
	exists, err := h.operations.CheckIfServiceExist(serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound,
			utils.FormatErrorResponse(fmt.Sprintf("service[ID:%d] doesn't exist", serviceID)))
		return
	}

	createdServiceVersion, err := h.operations.CreateServiceVersion(
		serviceID, serviceVersionToAdd[models.AttributeServiceVersionTag].(string),
		serviceVersionToAdd[models.AttributeServiceVersionInfo].(string))
	if err != nil {
		if err == appErrors.ErrServiceVersionAlreadyExists {
			c.JSON(http.StatusConflict,
				utils.FormatErrorResponse(
					fmt.Sprintf("Version %s already exists for service ID %d",
						serviceVersionToAdd[models.AttributeServiceVersionTag].(string), serviceID)))
		} else {
			c.JSON(http.StatusInternalServerError,
				utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		}
		return
	}
	c.JSON(http.StatusCreated, createdServiceVersion)
}

// updateServiceVersion update service version in DB.
// Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) updateServiceVersion(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var serviceID uint
	if _, err := fmt.Sscanf(id, "%d", &serviceID); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service ID should be numerical"))
		return
	}
	tag := c.Param(models.AttributeServiceVersionTag)
	if len(tag) == 0 {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Expected Version Tag as a part of URL"))
		return
	}

	var serviceVersionToUpdate map[string]interface{}
	if err := c.BindJSON(&serviceVersionToUpdate); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service Version update payload is invalid; Expected JSON payload"))
		return
	}

	if !utils.EnsureFieldsStrictlyExists(serviceVersionToUpdate, models.UpdateVersionPayloadTemplate) {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse(
				fmt.Sprintf("Service Version update payload is invalid; Strictly Allowed Params: %v",
					utils.ConvertFieldTypeToString(models.UpdateVersionPayloadTemplate))))
		return
	}

	exists, err := h.operations.CheckIfServiceExist(serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, utils.FormatErrorResponse(fmt.Sprintf("service[ID:%d] doesn't exist", serviceID)))
		return
	}
	updatedServiceVersion, err := h.operations.UpdateServiceVersion(
		serviceID, tag, serviceVersionToUpdate[models.AttributeServiceVersionInfo].(string))
	if err != nil {
		if err == appErrors.ErrServiceVersionDoesNotExist {
			c.JSON(http.StatusNotFound,
				utils.FormatErrorResponse(fmt.Sprintf("Service [ID:%d] with tag %s doesn't exist", serviceID, tag)))
		} else {
			c.JSON(http.StatusInternalServerError,
				utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, updatedServiceVersion)
}

// deleteServiceVersion deletes service version from DB
// Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) deleteServiceVersion(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var serviceID uint
	if _, err := fmt.Sscanf(id, "%d", &serviceID); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service ID should be numerical"))
		return
	}
	tag := c.Param(models.AttributeServiceVersionTag)
	if len(tag) == 0 {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Expected Non-empty Version Tag as a part of URL"))
		return
	}

	exists, err := h.operations.CheckIfServiceExist(serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound,
			utils.FormatErrorResponse(fmt.Sprintf("service[ID:%d] doesn't exist", serviceID)))
		return
	}

	err = h.operations.DeleteServiceVersion(serviceID, tag)
	if err != nil {
		if err == appErrors.ErrServiceVersionDoesNotExist {
			c.JSON(http.StatusNotFound,
				utils.FormatErrorResponse(
					fmt.Sprintf("Service [ID:%d] with tag %s doesn't exist", serviceID, tag)))
		} else {
			c.JSON(http.StatusInternalServerError,
				utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, utils.FormatGenericResponse("Service Version deleted from system"))
}

// fetchServiceVersions list the service versions in system
func (h *Handler) fetchServiceVersions(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var serviceID uint
	if _, err := fmt.Sscanf(id, "%d", &serviceID); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Service ID should be numerical"))
		return
	}

	pageStr := c.DefaultQuery("page", "0")
	page, paramErr := strconv.Atoi(pageStr)
	if paramErr != nil || page < 0 {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Request Path contains invalid page number, choose positive numerical value"))
		return
	}
	// page 0 or any unset page parameter, will represent the first page
	if page == 0 {
		page = 1
	}

	pageSizeStr := c.DefaultQuery("size", models.DefaultPageSize)

	pageSize, paramErr := strconv.Atoi(pageSizeStr)
	if paramErr != nil || pageSize < 0 {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Request Path contains invalid page size, choose positive numerical value"))
		return
	}

	exists, err := h.operations.CheckIfServiceExist(serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, utils.FormatErrorResponse(fmt.Sprintf("service[ID:%d] doesn't exist", serviceID)))
		return
	}

	users, total, err := h.operations.FetchServiceVersionsInverted(serviceID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	result := h.operations.FormatVersionDetailsWithPageDetails(users, total, page, pageSize)
	c.JSON(http.StatusOK, result)
}
