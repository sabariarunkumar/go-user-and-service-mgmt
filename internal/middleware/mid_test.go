package middleware

import (
	"net/http"
	"net/http/httptest"
	"userservice/internal/auth"
	"userservice/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Middleware Tests", func() {

	var router *gin.Engine
	gin.SetMode(gin.TestMode)
	Context("Test Authz middleware", func() {
		BeforeEach(func() {
			router = gin.Default()
		})
		It("user having expected role for the route in his request", func() {
			router.Use(func(c *gin.Context) {
				c.Set("role", "admin")
				c.Next()
			})
			router.GET("/test", AuthzRoles("admin", "basic"), func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Body.String()).To(Equal("OK"))
		})

		It("user having undesired role for the route in his request", func() {
			router.Use(func(c *gin.Context) {
				c.Set("role", "basic")
				c.Next()
			})
			router.GET("/test", AuthzRoles("admin"), func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring("user not authorized"))
		})

		It("user role is missing for the user request", func() {
			router.GET("/test", AuthzRoles("admin", "user"), func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring("user not authorized"))
		})
	})

	Context("Authenticate middleware", func() {
		var mockLog = zap.NewExample().Sugar()
		secret := []byte("secret")
		var token *string
		BeforeEach(func() {
			gin.SetMode(gin.TestMode)
			router = gin.Default()
			router.Use(Authenticate("", mockLog, secret))
			token, _ = auth.CreateJWT(secret, 5, "test@gmail.com", "admin")
			_ = token

		})

		It("ensure not authn for login request", func() {
			router.GET("/login", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			req, _ := http.NewRequest(http.MethodGet, "/login", nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Body.String()).To(Equal("OK"))

		})
		It("No Authorization header", func() {
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring(errors.ErrAuthzHeaderMissing.Error()))

		})
		It("Missing token", func() {
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring(errors.ErrMissingToken.Error()))

		})
		It("Invalid token", func() {
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer xyztoken")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring(errors.ErrInvalidOrExpiredToken.Error()))
		})
		It("Invalid or Expired token", func() {
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer xyztoken")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring(errors.ErrInvalidOrExpiredToken.Error()))
		})
		It("Only role claim present", func() {
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"role": "basic",
			})
			tokenString, _ := token.SignedString(secret)
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring(errors.ErrTokenClaimMissing.Error()))
		})
		It("Only email claim present", func() {
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"email": "p.sabari",
			})
			tokenString, _ := token.SignedString(secret)
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring(errors.ErrTokenClaimMissing.Error()))
		})
		It("Authenticated Request", func() {
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"role":  "basic",
				"email": "sabari@gmail.com",
			})
			tokenString, _ := token.SignedString(secret)
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Body.String()).To(ContainSubstring("OK"))
		})
	})

})
