package user

import (
	"fmt"
	"net/http"
	"strconv"
	"userservice/internal/auth"
	appErrors "userservice/internal/errors"
	"userservice/internal/misc"
	"userservice/internal/models"
	"userservice/internal/utils"

	"github.com/gin-gonic/gin"
)

// login validates payload and generate JWT token if its a successful login.
// Upon Successful login, if user has temporary password set,
// a flag[password_change_required] will be sent along.
// Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) login(c *gin.Context) {
	var userLogin map[string]interface{}
	if err := c.BindJSON(&userLogin); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Login payload is invalid; Expected JSON payload"))
		return
	}

	if !utils.EnsureFieldsStrictlyExists(userLogin, models.LoginPayloadTemplate) {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse(
			fmt.Sprintf("Login payload is invalid; Strictly Allowed Params: %v",
				utils.ConvertFieldTypeToString(models.LoginPayloadTemplate))))
		return
	}

	err := misc.PayloadValidator.Var(userLogin[models.AttributeEmail], "required,email")
	if err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("Login payload contains invalid email"))
		return
	}
	user, err := h.operations.GetUserByEmail(userLogin[models.AttributeEmail].(string))
	if err != nil {
		if err == appErrors.ErrInternal {
			c.JSON(http.StatusInternalServerError,
				utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
			return
		}
		// let us not explicitly inform about the user email not found,
		// such that non-legitimate users/hackers wont get an insight
		c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(appErrors.ErrInvalidEmailOrPass.Error()))
		return
	}

	if !auth.CompareHashAndPassword(user.PasswordHash, []byte(userLogin["password"].(string))) {
		c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(appErrors.ErrInvalidEmailOrPass.Error()))
		return
	}

	secret := []byte(h.runtimeConfig.JWTSecret)
	token, err := auth.CreateJWT(secret, h.runtimeConfig.JWTExpirationInSeconds, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	// Frontend will handle the response and forward it to change password endpoint if temp password not changed
	c.JSON(http.StatusOK, utils.FormatTokenResponse(*token, user.IsTemporaryPassword))
}

// getUserByID endpoint fetches user by ID
func (h *Handler) getUserByID(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var userId uint
	if _, err := fmt.Sscanf(id, "%d", &userId); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("User Fetch ID should be numerical"))
		return
	}

	user, err := h.operations.GetUser(userId)
	if err != nil {
		if err == appErrors.ErrInternal {
			c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
			return
		}
		c.JSON(http.StatusNotFound, utils.FormatErrorResponse(fmt.Sprintf("UserID %d doesn't exist", userId)))
		return
	}
	c.JSON(http.StatusOK, user)
}

// addUser adds new user to the system. Only admin users can add users.
// User will be created with temporary password, and shown to admin User.
// New user can login and change his password.
// Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) addUser(c *gin.Context) {
	var userToAdd map[string]interface{}
	if err := c.BindJSON(&userToAdd); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse("User creation payload is invalid; Expected JSON payload"))
		return
	}

	if !utils.EnsureFieldsStrictlyExists(userToAdd, models.RegisterOrUpdateUserPayloadTemplate) {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse(
			fmt.Sprintf("User creation payload is invalid; Strictly Allowed Params: %v",
				utils.ConvertFieldTypeToString(models.RegisterOrUpdateUserPayloadTemplate))))
		return
	}

	err := misc.PayloadValidator.Var(userToAdd[models.AttributeEmail], "required,email")
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse("User creation payload contains invalid email"))
		return
	}
	if _, ok := misc.Roles[userToAdd[models.AttributeRole].(string)]; !ok {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse(fmt.Sprintf("User Role %s doesn't exist", userToAdd[models.AttributeRole].(string))))
		return
	}

	temporaryPass, err := auth.GeneratePassword()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	temporaryPassHash, err := auth.GeneratePasswordHash(*temporaryPass)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}

	// let us consider email as the user name if not explicitly mentioned
	if len(userToAdd[models.AttributeName].(string)) == 0 {
		userToAdd[models.AttributeName] = userToAdd[models.AttributeEmail]
	}
	err = h.operations.CreateUser(userToAdd[models.AttributeName].(string),
		userToAdd[models.AttributeEmail].(string), userToAdd[models.AttributeRole].(string), temporaryPassHash)
	if err != nil {
		if err == appErrors.ErrUserWithSameEmailAlreadyExists {
			c.JSON(http.StatusConflict,
				utils.FormatErrorResponse(
					fmt.Sprintf("User with email %s already exists", userToAdd[models.AttributeEmail].(string))))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	// intimating the calling user about the temporary password being set for the newly added user
	c.JSON(http.StatusCreated, utils.FormatTempPassResponse(*temporaryPass))
}

// updateUser update user in DB. All the fields mentioned in RegisterOrUpdateUserPayloadTemplate needs to be part of
// user request. Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) updateUser(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var userId uint
	if _, err := fmt.Sscanf(id, "%d", &userId); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("User ID should be numerical"))
		return
	}
	var userToUpdate map[string]interface{}
	if err := c.BindJSON(&userToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse("User update payload is invalid; Expected JSON payload"))
		return
	}

	if !utils.EnsureFieldsStrictlyExists(userToUpdate, models.RegisterOrUpdateUserPayloadTemplate) {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse(fmt.Sprintf("User update payload is invalid; Strictly Allowed Params: %v",
			utils.ConvertFieldTypeToString(models.RegisterOrUpdateUserPayloadTemplate))))
		return
	}

	err := misc.PayloadValidator.Var(userToUpdate[models.AttributeEmail], "required,email")
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse("User update payload contains invalid email"))
		return
	}

	if _, ok := misc.Roles[userToUpdate[models.AttributeRole].(string)]; !ok || userToUpdate[models.AttributeRole] == "" {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse(fmt.Sprintf("User Role %s doesn't exist", userToUpdate[models.AttributeRole].(string))))
		return
	}

	updaterUser, err := h.operations.UpdateUser(userId, userToUpdate[models.AttributeName].(string),
		userToUpdate[models.AttributeEmail].(string), userToUpdate[models.AttributeRole].(string))
	if err != nil {
		if err == appErrors.ErrUserDoesNotExist {
			c.JSON(http.StatusNotFound, utils.FormatErrorResponse(appErrors.ErrUserDoesNotExist.Error()))
			return
		} else if err == appErrors.ErrUserWithSameEmailAlreadyExists {
			c.JSON(http.StatusConflict, utils.FormatErrorResponse(appErrors.ErrUserWithSameEmailAlreadyExists.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	// UI can reflect UserManagement page to get the updated users list
	c.JSON(http.StatusOK, updaterUser)
}

// deleteUser deletes user from system
// Request will be rejected if additional fields to desired ones are present in payload.
func (h *Handler) deleteUser(c *gin.Context) {
	id := c.Param(models.QueryParamID)
	var userId uint
	if _, err := fmt.Sscanf(id, "%d", &userId); err != nil {
		c.JSON(http.StatusBadRequest,
			utils.FormatErrorResponse("User ID should be numerical"))
		return
	}
	err := h.operations.DeleteUser(userId)
	if err != nil {
		if err == appErrors.ErrUserDoesNotExist {
			c.JSON(http.StatusNotFound, utils.FormatGenericResponse(appErrors.ErrUserDoesNotExist.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	// UI can reflect UserManagement page to get the updated users list
	c.JSON(http.StatusOK, utils.FormatGenericResponse("User deleted from system"))
}

// FetchServices list the users in system
func (h *Handler) fetchUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "0")

	page, paramErr := strconv.Atoi(pageStr)
	if paramErr != nil || page < 0 {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse("Payload contains invalid page number, choose positive numerical value"))
		return
	}

	// page 0 or unset page parameter will represent the first page
	if page == 0 {
		page = 1
	}
	pageSizeStr := c.DefaultQuery("size", models.DefaultPageSize)

	pageSize, paramErr := strconv.Atoi(pageSizeStr)
	if paramErr != nil || pageSize < 0 {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse("Payload contains invalid page size, choose positive numerical value"))
		return
	}

	users, total, err := h.operations.FetchUsersWithPagination(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	result := h.operations.FormatUserDetailsWithPageDetails(users, total, page, pageSize)
	c.JSON(http.StatusOK, result)
}

// ChangeUserPassword applies to authn user
func (h *Handler) changeUserPassword(c *gin.Context) {
	// email should be set by middleware, if not its considered as Internal error
	email, ok := c.Get(models.AttributeEmail)
	if !ok {
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	var userRequiringPassChange map[string]interface{}
	if err := c.BindJSON(&userRequiringPassChange); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse("User password change payload is invalid; Expected JSON payload"))
		return
	}

	if !utils.EnsureFieldsStrictlyExists(userRequiringPassChange, models.PasswordChangePayloadTemplate) {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse(
			fmt.Sprintf("User password change payload is invalid; Strictly Allowed Params: %v",
				utils.ConvertFieldTypeToString(models.PasswordChangePayloadTemplate))))
		return
	}

	if userRequiringPassChange["password"] == "" {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse(appErrors.ErrPasswordMissingOrEmpty.Error()))
		return
	}

	// Hash generation accepts password of length <= 72, hence having this upper limit.
	if len(userRequiringPassChange["password"].(string)) > 72 {
		c.JSON(http.StatusBadRequest, utils.FormatErrorResponse(appErrors.ErrPasswordTooLong.Error()))
		return
	}

	passwordHash, err := auth.GeneratePasswordHash(userRequiringPassChange["password"].(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	// Email as well serves as a unique id to user
	err = h.operations.ChangePassword(email.(string), passwordHash)
	if err != nil {
		if err == appErrors.ErrUserDoesNotExist {
			c.JSON(http.StatusNotFound, utils.FormatErrorResponse(appErrors.ErrUserDoesNotExist.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.FormatErrorResponse(appErrors.ErrFailureToProcessRequest.Error()))
		return
	}
	// UI can reflect UserManagement page to get the updated users list
	c.Status(http.StatusOK)
}
