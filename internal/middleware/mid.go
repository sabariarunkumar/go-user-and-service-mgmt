package middleware

import (
	"net/http"
	"strings"
	"userservice/internal/auth"
	"userservice/internal/errors"
	"userservice/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// Authenticate validates JWT Token and checks for existence of desired claims
func Authenticate(apiPrefix string, log *zap.SugaredLogger, secret []byte) gin.HandlerFunc {

	return func(c *gin.Context) {

		// skip Token validation for login endpoint
		if c.Request.URL != nil && (c.Request.URL.Path == apiPrefix+"/login") {
			c.Next()
			return
		}
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(errors.ErrAuthzHeaderMissing.Error()))
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(errors.ErrMissingToken.Error()))
			c.Abort()
			return
		}
		tokenString := parts[1]

		token, err := auth.ValidateJWT(secret, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(errors.ErrInvalidOrExpiredToken.Error()))
			c.Abort()
			return
		}
		if !token.Valid {
			c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(errors.ErrInvalidOrExpiredToken.Error()))
			c.Abort()
			return
		}
		// let us generically say token lacks claim headers, as we know its tampered at this point
		// If we Specify which claim exactly, hackers will get insight.
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(errors.ErrTokenClaimMissing.Error()))
			c.Abort()
			return
		}
		role, ok := claims[auth.JWTClaimRole].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(errors.ErrTokenClaimMissing.Error()))
			c.Abort()
			return
		}
		email, ok := claims[auth.JWTClaimEmail].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(errors.ErrTokenClaimMissing.Error()))
			c.Abort()
			return
		}

		// set the parameters for endpoints to access
		c.Set(auth.JWTClaimRole, role)
		c.Set(auth.JWTClaimEmail, email)
		c.Next()
	}
}

// AuthzRoles middleware checks if the user request comply with associated endpoint request roles
func AuthzRoles(allowedRolesForRoute ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, ok := c.Get(auth.JWTClaimRole)
		if !ok {
			c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(errors.ErrUserNotAuthorized.Error()))
			c.Abort()
			return
		}
		for _, role := range allowedRolesForRoute {
			if role == userRole {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusUnauthorized, utils.FormatErrorResponse(errors.ErrUserNotAuthorized.Error()))
		c.Abort()
	}
}
