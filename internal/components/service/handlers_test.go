package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"userservice/internal/configs"
	appErrors "userservice/internal/errors"
	"userservice/internal/misc"
	"userservice/internal/models"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func GetTestGinContext(w *httptest.ResponseRecorder) *gin.Context {
	gin.SetMode(gin.TestMode)

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}
	return ctx
}
func MockJsonPostOrPut(c *gin.Context, content interface{}) {
	c.Request.Header.Set("Content-Type", "application/json")

	jsonbytes, err := json.Marshal(content)
	if err != nil {
		panic(err)
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonbytes))
}

var _ = Describe("Services", func() {

	var (
		ctx                   *gin.Context
		handler               *Handler
		w                     *httptest.ResponseRecorder
		u                     url.Values
		operationsWithoutErr  ServiceAndVersionMock
		operationsInternalErr = ServiceAndVersionMock{
			SetInternalError: MockFuncs{
				ServiceExistenceFn:        struct{}{},
				ServiceVersionExistenceFn: struct{}{},
				GetServiceFn:              struct{}{},
				CreateServiceFn:           struct{}{},
				UpdateServiceFn:           struct{}{},
				DeleteServiceFn:           struct{}{},
				FetchServiceFn:            struct{}{},
			},
		}
		operationsRecordNotFound = ServiceAndVersionMock{
			SetRecordNotFound: MockFuncs{
				ServiceExistenceFn: struct{}{},
				GetServiceFn:       struct{}{},
				UpdateServiceFn:    struct{}{},
				DeleteServiceFn:    struct{}{},
			},
		}
		operationsDuplicateEntity = ServiceAndVersionMock{
			SetRecordAlreadyExist: MockFuncs{
				CreateServiceFn: struct{}{},
				UpdateServiceFn: struct{}{},
			},
		}
	)
	BeforeEach(func() {
		handler = new(Handler)
		handler.runtimeConfig = new(configs.Config)
		w = httptest.NewRecorder()
		ctx = GetTestGinContext(w)
		misc.InitPayloadValidator()
		operationsWithoutErr = ServiceAndVersionMock{}

		u = url.Values{}
	})
	Context("getServiceByID", func() {
		It("invalid/Non-numerical path param ID", func() {

			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.getServiceByID(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service Fetch ID should be numerical"))
		})
		It("DB Internal error", func() {

			handler.operations = &operationsInternalErr
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.getServiceByID(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("DB Record not found", func() {

			handler.operations = &operationsRecordNotFound
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.getServiceByID(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring("doesn't exist"))
		})
		It("Successful fetch", func() {
			var service models.Service
			service.ID = 1
			operationsWithoutErr.Service = &service
			handler.operations = &operationsWithoutErr
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.getServiceByID(ctx)
			Expect(w.Code).To(Equal(200))
			var recvService models.Service
			err := json.Unmarshal(w.Body.Bytes(), &recvService)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(recvService.ID).To(Equal(recvService.ID))
		})
	})
	Context("addService", func() {
		It("Invalid payload", func() {
			handler.operations = &operationsWithoutErr
			handler.addService(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service creation payload is invalid; Expected JSON payload"))
		})
		It("Missing service name or description in payload", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addService(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service creation payload is invalid; Strictly Allowed Params:"))
		})
		It("empty service name", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addService(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service Metadata Creation payload is invalid: service name is empty"))
		})

		It("DB Internal error", func() {
			handler.operations = &operationsInternalErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "postman",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addService(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})

		It("Duplicate same name exist", func() {
			handler.operations = &operationsDuplicateEntity
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "postman",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addService(ctx)
			Expect(w.Code).To(Equal(409))
			Expect(w.Body.String()).To(ContainSubstring("Service postman already exists"))
		})

		It("successful service creation", func() {
			var service models.Service
			service.Name = "postman-v2"
			operationsWithoutErr.Service = &service
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        service.Name,
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addService(ctx)
			Expect(w.Code).To(Equal(201))
			var recvService models.Service
			err := json.Unmarshal(w.Body.Bytes(), &recvService)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(recvService.Name).To(Equal(recvService.Name))
			Expect(w.Body.String()).To(Not(BeEmpty()))
		})
	})
	Context("updateService", func() {

		It("invalid/Non-numerical path param ID", func() {

			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.updateService(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service ID should be numerical"))
		})

		It("Invalid payload", func() {
			handler.operations = &operationsWithoutErr
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateService(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service Metadata update payload is invalid; Expected JSON payload"))
		})
		It("Missing service name in payload", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateService(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service Metadata update payload is invalid; Strictly Allowed Params"))
		})
		It("empty service name", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateService(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service Metadata update payload is invalid: service name is empty"))
		})

		It("already same service name exist which current user is interested to change into", func() {
			handler.operations = &operationsDuplicateEntity
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "postman",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateService(ctx)
			Expect(w.Code).To(Equal(409))
			Expect(w.Body.String()).To(ContainSubstring("Service postman already exists"))
		})

		It("DB Internal Error", func() {
			handler.operations = &operationsInternalErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "postman",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateService(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})

		It("service doesn't exist", func() {
			handler.operations = &operationsRecordNotFound
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "postman",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateService(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring("Service postman doesn't exists"))
		})
		It("Successful Update request", func() {
			var service models.Service
			service.Name = "postman-v2"
			operationsWithoutErr.Service = &service
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        service.Name,
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateService(ctx)
			var recvService models.Service
			err := json.Unmarshal(w.Body.Bytes(), &recvService)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(w.Code).To(Equal(200))
			Expect(recvService.Name).To(Equal(recvService.Name))
		})
	})

	Context("deleteService", func() {
		It("invalid/Non-numerical path param ID", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.deleteService(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service ID should be numerical"))
		})
		It("DB Internal Error", func() {
			handler.operations = &operationsInternalErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "postman",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.deleteService(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})

		It("service doesn't exist", func() {
			handler.operations = &operationsRecordNotFound
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "postman",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.deleteService(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring("Service[ID:1] doesn't exist"))
		})
		It("successful deletion request", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":        "postman",
				"description": "Support scalability",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.deleteService(ctx)
			Expect(w.Code).To(Equal(200))
			Expect(w.Body.String()).To(ContainSubstring("Service deleted from system"))
		})
	})
	Context("fetchServices", func() {
		It("Invalid page param [non numerical]", func() {
			u.Add("page", "invalid")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServices(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid page number, choose positive numerical value"))
		})
		It("Invalid page param [negative]", func() {
			u.Add("page", "-1")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServices(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid page number, choose positive numerical value"))
		})
		It("Invalid size param [non numerical]", func() {
			u.Add("page", "0")
			u.Add("size", "invalid")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServices(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid page size, choose positive numerical value"))
		})
		It("Invalid size param [negative]", func() {
			u.Add("page", "0")
			u.Add("size", "-1")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServices(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid page size, choose positive numerical value"))
		})
		It("Invalid sort_by param [other than name or date]", func() {
			u.Add("page", "0")
			u.Add("size", "0")
			u.Add("sort_by", "versions")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServices(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid sort_by value, choose name or date"))
		})
		It("Invalid inverted param [other than true or false]", func() {
			u.Add("page", "0")
			u.Add("size", "0")
			u.Add("sort_by", "name")
			u.Add("inverted", "default")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServices(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid inverted value, choose true or false"))
		})
		It("DB Internal Error", func() {
			u.Add("page", "0")
			u.Add("size", "0")
			u.Add("sort_by", "name")
			u.Add("inverted", "true")
			handler.operations = &operationsInternalErr
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServices(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).
				To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("successful fetch request", func() {
			u.Add("page", "0")
			u.Add("size", "0")
			u.Add("sort_by", "name")
			u.Add("inverted", "true")
			var service models.Service
			service.Name = "postman"
			operationsWithoutErr.Service = &service
			handler.operations = &operationsWithoutErr
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServices(ctx)
			Expect(w.Code).To(Equal(200))
			var recvService models.PaginatedServiceList
			err := json.Unmarshal(w.Body.Bytes(), &recvService)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(recvService.Data[0].Name).To(Equal(service.Name))
		})

	})

})

var _ = Describe("Service versions", func() {

	var (
		ctx        *gin.Context
		handler    *Handler
		w          *httptest.ResponseRecorder
		u          url.Values
		operations ServiceAndVersionMock
	)
	BeforeEach(func() {
		handler = new(Handler)
		handler.runtimeConfig = new(configs.Config)
		w = httptest.NewRecorder()
		ctx = GetTestGinContext(w)
		misc.InitPayloadValidator()
		operations = ServiceAndVersionMock{}
		u = url.Values{}
	})
	Context("getServiceVersionByID", func() {
		It("invalid/Non-numerical path param Service ID", func() {

			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.getServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service ID should be numerical"))
		})
		It("Empty path param Service Tag", func() {
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "0",
				},
				{
					Key:   "tag",
					Value: "",
				},
			}
			ctx.Params = params
			handler.getServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Expected Non-empty Version Tag as a part of URL"))
		})
		It("DB Internal error while checking if service ID is valid", func() {
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.getServiceVersion(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).
				To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("DB Record not found while checking if service ID is valid", func() {
			operations = ServiceAndVersionMock{
				SetRecordNotFound: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.getServiceVersion(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).
				To(ContainSubstring("service[ID:1] doesn't exist"))
		})
		It("DB Internal error while fetching service version", func() {
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					GetServiceVersionFn: struct{}{},
				},
			}
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.getServiceVersion(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).
				To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("Corresponding Service table DB Record not found", func() {
			operations = ServiceAndVersionMock{
				SetRecordNotFound: MockFuncs{
					GetServiceVersionFn: struct{}{},
				},
			}
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.getServiceVersion(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).
				To(ContainSubstring("version tag v1 doesn't exist for service ID 1"))
		})
		It("Successful Record Fetch", func() {
			var serviceVersion models.ServiceVersion
			serviceVersion.Info = "VERSION"
			operations.Version = &serviceVersion
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.getServiceVersion(ctx)
			Expect(w.Code).To(Equal(200))
			var recvServiceVersion models.ServiceVersion
			err := json.Unmarshal(w.Body.Bytes(), &recvServiceVersion)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(recvServiceVersion.Info).To(Equal(serviceVersion.Info))
		})
	})
	Context("addServiceVersion", func() {
		It("invalid/Non-numerical path param Service ID", func() {

			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.addServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Service ID should be numerical"))
		})
		It("Invalid payload", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.operations = &operations
			handler.addServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Service Version creation payload is invalid; Expected JSON payload"))
		})
		It("Missing service version tag or info in payload", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"info": "version-1",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Service Version creation payload is invalid; Strictly Allowed Params:"))
		})
		It("empty service version tag", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"tag":  "",
				"info": "version-1",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Service Version creation payload is invalid: version tag is empty"))
		})
		It("DB Internal error while checking if service ID is valid", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"tag":  "postman",
				"info": "version-1",
			}
			MockJsonPostOrPut(ctx, payload)
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			handler.addServiceVersion(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).
				To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("DB Record not found while checking if service ID is valid", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"tag":  "postman",
				"info": "version-1",
			}
			MockJsonPostOrPut(ctx, payload)
			operations = ServiceAndVersionMock{
				SetRecordNotFound: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			handler.addServiceVersion(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).
				To(ContainSubstring("service[ID:1] doesn't exist"))
		})

		It("DB Record already exist with same name", func() {
			operations = ServiceAndVersionMock{
				SetRecordAlreadyExist: MockFuncs{
					CreateServiceVersionFn: struct{}{},
				},
			}
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"tag":  "postman",
				"info": "version-1",
			}
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			MockJsonPostOrPut(ctx, payload)
			handler.addServiceVersion(ctx)
			Expect(w.Code).To(Equal(409))
			Expect(w.Body.String()).
				To(ContainSubstring("Version postman already exists for service ID 1"))
		})

		It("DB Internal error while creating service version", func() {
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					CreateServiceVersionFn: struct{}{},
				},
			}
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"tag":  "postman",
				"info": "version-1",
			}
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			MockJsonPostOrPut(ctx, payload)
			handler.addServiceVersion(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).
				To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("Successful creation of service version", func() {
			var version models.ServiceVersion
			version.Tag = "v1"
			operations.Version = &version
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"tag":  version.Tag,
				"info": "version-1",
			}
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			MockJsonPostOrPut(ctx, payload)
			handler.addServiceVersion(ctx)
			Expect(w.Code).To(Equal(201))
			var recvServiceVersion models.ServiceVersion
			err := json.Unmarshal(w.Body.Bytes(), &recvServiceVersion)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(recvServiceVersion.Tag).To(Equal(version.Tag))
		})
	})
	Context("updateServiceVersion", func() {
		It("invalid/Non-numerical path param Service ID", func() {

			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.updateServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service ID should be numerical"))
		})
		It("empty service version tag", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			handler.updateServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Expected Version Tag as a part of URL"))
		})
		It("Invalid payload", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.operations = &operations
			handler.updateServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service Version update payload is invalid; Expected JSON payload"))
		})
		It("Missing service version info in payload", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{}
			MockJsonPostOrPut(ctx, payload)
			handler.updateServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service Version update payload is invalid; Strictly Allowed Params:"))
		})

		It("DB Internal error while checking if service ID is valid", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"info": "updated",
			}
			MockJsonPostOrPut(ctx, payload)
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			handler.updateServiceVersion(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("DB Record not found while checking if service ID is valid", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"info": "updated",
			}
			MockJsonPostOrPut(ctx, payload)
			operations = ServiceAndVersionMock{
				SetRecordNotFound: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			handler.updateServiceVersion(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring("service[ID:1] doesn't exist"))
		})

		It("DB Record with given version tag doesn't exist", func() {
			operations = ServiceAndVersionMock{
				SetRecordNotFound: MockFuncs{
					UpdateServiceVersionFn: struct{}{},
				},
			}
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"info": "updated",
			}
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			MockJsonPostOrPut(ctx, payload)
			handler.updateServiceVersion(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring("Service [ID:1] with tag v1 doesn't exist"))
		})
		It("DB Record with given version tag doesn't exist", func() {
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					UpdateServiceVersionFn: struct{}{},
				},
			}
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"info": "updated",
			}
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			MockJsonPostOrPut(ctx, payload)
			handler.updateServiceVersion(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("Successful update", func() {
			var version models.ServiceVersion
			version.Tag = "v1"
			operations.Version = &version
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"info": "updated",
			}
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			MockJsonPostOrPut(ctx, payload)
			handler.updateServiceVersion(ctx)
			Expect(w.Code).To(Equal(200))
			var recvServiceVersion models.ServiceVersion
			err := json.Unmarshal(w.Body.Bytes(), &recvServiceVersion)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(recvServiceVersion.Tag).To(Equal(version.Tag))
		})

	})
	Context("deleteServiceVersion", func() {
		It("invalid/Non-numerical path param Service ID", func() {

			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.deleteServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service ID should be numerical"))
		})
		It("Empty path param Service Tag", func() {
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "0",
				},
				{
					Key:   "tag",
					Value: "",
				},
			}
			ctx.Params = params
			handler.deleteServiceVersion(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(
				ContainSubstring("Expected Non-empty Version Tag as a part of URL"))
		})
		It("DB Internal error while checking if service ID is valid", func() {
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.deleteServiceVersion(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(
				ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("DB Record not found while checking if service ID is valid", func() {
			operations = ServiceAndVersionMock{
				SetRecordNotFound: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.deleteServiceVersion(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(
				ContainSubstring("service[ID:1] doesn't exist"))
		})
		It("DB Record with given version tag doesn't exist", func() {
			operations = ServiceAndVersionMock{
				SetRecordNotFound: MockFuncs{
					DeleteServiceVersionFn: struct{}{},
				},
			}
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")

			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.deleteServiceVersion(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring("Service [ID:1] with tag v1 doesn't exist"))
		})
		It("DB Record with given version tag doesn't exist", func() {
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					DeleteServiceVersionFn: struct{}{},
				},
			}
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")

			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.deleteServiceVersion(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("Successful update", func() {
			handler.operations = &operations
			ctx.Request.Header.Set("Content-Type", "application/json")

			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
				{
					Key:   "tag",
					Value: "v1",
				},
			}
			ctx.Params = params
			handler.deleteServiceVersion(ctx)
			Expect(w.Code).To(Equal(200))
		})

	})
	Context("fetchServiceVersions", func() {
		It("invalid/Non-numerical path param Service ID", func() {

			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.fetchServiceVersions(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Service ID should be numerical"))
		})
		It("Invalid page param [non numerical]", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			u.Add("page", "invalid")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServiceVersions(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid page number, choose positive numerical value"))
		})
		It("Invalid page param [negative]", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			u.Add("page", "-1")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServiceVersions(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid page number, choose positive numerical value"))
		})
		It("Invalid size param [non numerical]", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			u.Add("page", "0")
			u.Add("size", "invalid")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServiceVersions(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid page size, choose positive numerical value"))
		})
		It("Invalid size param [negative]", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			u.Add("page", "0")
			u.Add("size", "-1")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchServiceVersions(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).
				To(ContainSubstring("Request Path contains invalid page size, choose positive numerical value"))
		})
		It("DB Internal error while checking if service ID is valid", func() {
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			ctx.Params = params
			u.Add("page", "0")
			u.Add("size", "1")
			handler.fetchServiceVersions(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).
				To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("DB Record not found while checking if service ID is valid", func() {
			operations = ServiceAndVersionMock{
				SetRecordNotFound: MockFuncs{
					ServiceExistenceFn: struct{}{},
				},
			}
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			u.Add("page", "0")
			u.Add("size", "1")
			ctx.Params = params
			handler.fetchServiceVersions(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).
				To(ContainSubstring("service[ID:1] doesn't exist"))
		})
		It("DB Record not found while checking if service ID is valid", func() {
			operations = ServiceAndVersionMock{
				SetInternalError: MockFuncs{
					FetchServiceVersionFn: struct{}{},
				},
			}
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			u.Add("page", "0")
			u.Add("size", "1")
			ctx.Params = params
			handler.fetchServiceVersions(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).
				To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("successful fetch request", func() {
			var serviceVersion models.ServiceVersion
			serviceVersion.Tag = "v1"
			operations.Version = &serviceVersion
			handler.operations = &operations
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			u.Add("page", "0")
			u.Add("size", "1")
			ctx.Params = params
			handler.fetchServiceVersions(ctx)
			Expect(w.Code).To(Equal(200))
			var recvService models.PaginatedVersionList
			err := json.Unmarshal(w.Body.Bytes(), &recvService)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(recvService.Data[0].Tag).To(Equal(serviceVersion.Tag))
		})
	})

})
