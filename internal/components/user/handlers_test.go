package user

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

var _ = Describe("Users", func() {

	var (
		ctx                         *gin.Context
		handler                     *Handler
		w                           *httptest.ResponseRecorder
		u                           url.Values
		operationsWithoutErr        UserMock
		operationsInternalErr       = UserMock{SetInternalError: true}
		operationsEmailOrIDNotFound = UserMock{SetEmailOrIDNotFound: true}
		operationsDuplicateEmail    = UserMock{SetDuplicateEmail: true}
		operationsUserDoesntExist   = UserMock{SetUserDoesntExist: true}
	)
	BeforeEach(func() {
		handler = new(Handler)
		handler.runtimeConfig = new(configs.Config)
		handler.runtimeConfig.JWTSecret = "mgmtportal"
		handler.runtimeConfig.JWTExpirationInSeconds = 100
		w = httptest.NewRecorder()
		ctx = GetTestGinContext(w)
		misc.InitPayloadValidator()
		operationsWithoutErr = UserMock{}
		misc.Roles["basic"] = ""
		u = url.Values{}

	})
	Context("Login", func() {
		It("Invalid payload", func() {
			handler.operations = &operationsWithoutErr
			handler.login(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Login payload is invalid; Expected JSON payload"))
		})
		It("Missing password or email in payload", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var loginPayload = map[string]interface{}{
				"email": "admin@mgmtportal.com",
			}
			MockJsonPostOrPut(ctx, loginPayload)
			handler.login(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Login payload is invalid; Strictly Allowed Params:"))
		})
		It("Invalid email format", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var loginPayload = map[string]interface{}{
				"email":    "admin",
				"password": "pass",
			}
			MockJsonPostOrPut(ctx, loginPayload)
			handler.login(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Login payload contains invalid email"))
		})
		It("DB Internal error", func() {
			handler.operations = &operationsInternalErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var loginPayload = map[string]interface{}{
				"email":    "admin@mgmtportal.com",
				"password": "pass",
			}
			MockJsonPostOrPut(ctx, loginPayload)
			handler.login(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("DB Record not found for email", func() {
			handler.operations = &operationsEmailOrIDNotFound
			ctx.Request.Header.Set("Content-Type", "application/json")
			var loginPayload = map[string]interface{}{
				"email":    "admin@mgmtportal.com",
				"password": "pass",
			}
			MockJsonPostOrPut(ctx, loginPayload)
			handler.login(ctx)
			Expect(w.Code).To(Equal(401))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrInvalidEmailOrPass.Error()))
		})
		It("password is wrong", func() {
			operationsWithoutErr.User = new(models.User)
			operationsWithoutErr.User.PasswordHash = "$2a$10$MMMx.hCq9QXeJyOm80Cx3e0o0PR25/xF05WgM9CsJR6zlnfbllZR2"
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var loginPayload = map[string]interface{}{
				"email":    "admin@mgmtportal.com",
				"password": "wrongPass",
			}
			MockJsonPostOrPut(ctx, loginPayload)
			handler.login(ctx)
			Expect(w.Code).To(Equal(401))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrInvalidEmailOrPass.Error()))
		})
		It("successful login", func() {
			operationsWithoutErr.User = new(models.User)
			operationsWithoutErr.User.PasswordHash = "$2a$10$MMMx.hCq9QXeJyOm80Cx3e0o0PR25/xF05WgM9CsJR6zlnfbllZR2"
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var loginPayload = map[string]interface{}{
				"email":    "admin@mgmtportal.com",
				"password": "sabari123",
			}
			MockJsonPostOrPut(ctx, loginPayload)
			handler.login(ctx)
			Expect(w.Code).To(Equal(200))
			Expect(w.Body.String()).To(Not(BeEmpty()))
		})
	})
	Context("getUserByID", func() {
		It("invalid/Non-numerical path param ID", func() {

			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.getUserByID(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User Fetch ID should be numerical"))
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
			handler.getUserByID(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("DB Record not found", func() {

			handler.operations = &operationsEmailOrIDNotFound
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.getUserByID(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring("doesn't exist"))
		})
		It("Successful fetch", func() {
			var user models.User
			user.ID = 1
			operationsWithoutErr.User = &user
			handler.operations = &operationsWithoutErr
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.getUserByID(ctx)
			Expect(w.Code).To(Equal(200))
			var recvUser models.User
			err := json.Unmarshal(w.Body.Bytes(), &recvUser)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(recvUser.ID).To(Equal(user.ID))
		})
	})
	Context("addUser", func() {
		It("Invalid payload", func() {
			handler.operations = &operationsWithoutErr
			handler.addUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User creation payload is invalid; Expected JSON payload"))
		})
		It("Missing name or email or password in payload", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"email": "admin@mgmtportal.com",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User creation payload is invalid; Strictly Allowed Params:"))
		})
		It("Invalid email format", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User creation payload contains invalid email"))
		})
		It("Invalid role", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "unknown",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User Role unknown doesn't exist"))
		})
		It("DB Internal error", func() {

			handler.operations = &operationsInternalErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addUser(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})

		It("user with same email exist", func() {
			handler.operations = &operationsDuplicateEmail
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addUser(ctx)
			Expect(w.Code).To(Equal(409))
			Expect(w.Body.String()).To(ContainSubstring("User with email admin@gmail.com already exists"))
		})

		It("successful user creation", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "",
				"email": "admin@gmail.com",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			handler.addUser(ctx)
			Expect(w.Code).To(Equal(201))
			Expect(w.Body.String()).To(Not(BeEmpty()))
		})
	})
	Context("updateUser", func() {

		It("invalid/Non-numerical path param ID", func() {

			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.updateUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User ID should be numerical"))
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
			handler.updateUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User update payload is invalid; Expected JSON payload"))
		})
		It("Missing name or email or password in payload", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"email": "admin@mgmtportal.com",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User update payload is invalid; Strictly Allowed Params"))
		})
		It("Invalid email format", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User update payload contains invalid email"))
		})
		It("Invalid role", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "unknown",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User Role unknown doesn't exist"))
		})

		It("user with same email exist", func() {
			handler.operations = &operationsDuplicateEmail
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateUser(ctx)
			Expect(w.Code).To(Equal(409))
			Expect(w.Body.String()).To(ContainSubstring("user already exists with same email"))
		})

		It("DB Internal Error", func() {
			handler.operations = &operationsInternalErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateUser(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})

		It("user doesn't exist", func() {
			handler.operations = &operationsUserDoesntExist
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateUser(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrUserDoesNotExist.Error()))
		})
		It("Successful Update request", func() {
			var user models.User
			user.Email = "adminv2@gmail.com"
			operationsWithoutErr.User = &user
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": user.Email,
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.updateUser(ctx)
			var recvUser models.User
			err := json.Unmarshal(w.Body.Bytes(), &recvUser)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(w.Code).To(Equal(200))
			Expect(recvUser.Email).To(Equal(user.Email))
		})
	})
	Context("deleteUser", func() {

		It("invalid/Non-numerical path param ID", func() {
			params := []gin.Param{
				{
					Key:   "id",
					Value: "invalid",
				},
			}
			ctx.Params = params
			handler.deleteUser(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User ID should be numerical"))
		})
		It("DB Internal Error", func() {
			handler.operations = &operationsInternalErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.deleteUser(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})

		It("user doesn't exist", func() {
			handler.operations = &operationsUserDoesntExist
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.deleteUser(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrUserDoesNotExist.Error()))
		})
		It("successful deletion request", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"name":  "admin",
				"email": "admin@gmail.com",
				"role":  "basic",
			}
			MockJsonPostOrPut(ctx, payload)
			params := []gin.Param{
				{
					Key:   "id",
					Value: "1",
				},
			}
			ctx.Params = params
			handler.deleteUser(ctx)
			Expect(w.Code).To(Equal(200))
			Expect(w.Body.String()).To(ContainSubstring("User deleted from system"))
		})
	})
	Context("fetchUsers", func() {
		It("Invalid page param [non numerical]", func() {
			u.Add("page", "invalid")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchUsers(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Payload contains invalid page number, choose positive numerical value"))
		})
		It("Invalid page param [negative]", func() {
			u.Add("page", "-1")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchUsers(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Payload contains invalid page number, choose positive numerical value"))
		})
		It("Invalid size param [non numerical]", func() {
			u.Add("page", "0")
			u.Add("size", "invalid")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchUsers(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Payload contains invalid page size, choose positive numerical value"))
		})
		It("Invalid size param [negative]", func() {
			u.Add("page", "0")
			u.Add("size", "-1")
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchUsers(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("Payload contains invalid page size, choose positive numerical value"))
		})
		It("DB Internal Error", func() {
			u.Add("page", "0")
			u.Add("size", "0")
			handler.operations = &operationsInternalErr
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchUsers(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("successful fetch request", func() {
			u.Add("page", "0")
			u.Add("size", "0")
			var user models.User
			user.Email = "adminv2@gmail.com"
			operationsWithoutErr.User = &user
			handler.operations = &operationsWithoutErr
			ctx.Request.URL.RawQuery = u.Encode()
			handler.fetchUsers(ctx)
			Expect(w.Code).To(Equal(200))
			var recvUser models.PaginatedUserList
			err := json.Unmarshal(w.Body.Bytes(), &recvUser)
			if err != nil {
				Fail(fmt.Sprintf("Internal error: %v", err))
			}
			Expect(recvUser.Data[0].Email).To(Equal(user.Email))
		})

	})
	Context("changeUserPassword", func() {

		It("email context not set", func() {
			handler.operations = &operationsWithoutErr
			handler.changeUserPassword(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))
		})
		It("Invalid payload", func() {
			handler.operations = &operationsWithoutErr
			ctx.Set("email", "admin@mgmtportal.com")
			handler.changeUserPassword(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User password change payload is invalid; Expected JSON payload"))
		})

		It("Missing password in payload", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{}
			MockJsonPostOrPut(ctx, payload)
			ctx.Set("email", "admin@mgmtportal.com")
			handler.changeUserPassword(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring("User password change payload is invalid; Strictly Allowed Params"))
		})
		It("empty Password", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"password": "",
			}
			MockJsonPostOrPut(ctx, payload)
			ctx.Set("email", "admin@mgmtportal.com")
			handler.changeUserPassword(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrPasswordMissingOrEmpty.Error()))
		})
		It("Password length more than 72", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"password": "sqlmock is a mock library implementing sql/driver." +
					"Which has one and only purpose - to simulate any sql driver" +
					"behavior in tests, without needing a real database connection. It helps to maintain correct TDD workflow",
			}
			MockJsonPostOrPut(ctx, payload)
			ctx.Set("email", "admin@mgmtportal.com")
			handler.changeUserPassword(ctx)
			Expect(w.Code).To(Equal(400))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrPasswordTooLong.Error()))
		})
		It("Successful change request", func() {
			handler.operations = &operationsWithoutErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"password": "admin123",
			}
			MockJsonPostOrPut(ctx, payload)
			ctx.Set("email", "admin@mgmtportal.com")
			handler.changeUserPassword(ctx)
			Expect(w.Code).To(Equal(200))
		})
		It("DB Internal error", func() {
			handler.operations = &operationsInternalErr
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"password": "admin123",
			}
			MockJsonPostOrPut(ctx, payload)
			ctx.Set("email", "admin@mgmtportal.com")
			handler.changeUserPassword(ctx)
			Expect(w.Code).To(Equal(500))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrFailureToProcessRequest.Error()))

		})
		It("User doesn't exist", func() {
			handler.operations = &operationsUserDoesntExist
			ctx.Request.Header.Set("Content-Type", "application/json")
			var payload = map[string]interface{}{
				"password": "admin123",
			}
			MockJsonPostOrPut(ctx, payload)
			ctx.Set("email", "admin@mgmtportal.com")
			handler.changeUserPassword(ctx)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(ContainSubstring(appErrors.ErrUserDoesNotExist.Error()))

		})
	})
})
