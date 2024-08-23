package user

import (
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("User [Handler]", func() {

	It("Initializer Handler, list and ensure expected number of routes", func() {
		h := NewHandler(nil, nil, nil)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		apiGroup := router.Group("/api/v1")
		h.RegisterRoutes(apiGroup)
		routes := router.Routes()
		Expect(routes).To(HaveLen(7))
	})
})
