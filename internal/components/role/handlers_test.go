package role

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"userservice/internal/errors"
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

var _ = Describe("Fetch Role [Handler]", func() {

	var (
		ctx     *gin.Context
		handler *Handler
		w       *httptest.ResponseRecorder
	)
	BeforeEach(func() {
		handler = new(Handler)
		w = httptest.NewRecorder()
		ctx = GetTestGinContext(w)
	})

	It("Successful Fetch", func() {
		var roleMockDefaultStruct RoleMockDefault
		handler.operations = &roleMockDefaultStruct
		handler.fetchUserRoles(ctx)
		Expect(w.Code).To(Equal(200))
		expectedRoles, _ := roleMockDefaultStruct.FetchRoles()

		var receivedRoles []models.UserRole
		err := json.Unmarshal(w.Body.Bytes(), &receivedRoles)
		if err != nil {
			Fail(fmt.Sprintf("Internal error: %v", err))
		}
		for _, expectedRole := range expectedRoles {
			var isPresent bool
			for _, receivedRole := range receivedRoles {
				if receivedRole.Name == expectedRole.Name && receivedRole.Description == expectedRole.Description {
					isPresent = true
				}
			}
			Expect(isPresent).To(BeTrue())
		}
	})
	It("Facing DB errors", func() {
		handler.operations = new(RoleMockForcedError)
		handler.fetchUserRoles(ctx)
		Expect(w.Code).To(Equal(500))
		Expect(w.Body.String()).To(ContainSubstring(errors.ErrFailureToProcessRequest.Error()))
	})
})
