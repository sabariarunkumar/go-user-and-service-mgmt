package service

import (
	"context"
	"regexp"
	"sync"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _ = Describe("Service [Handler]", func() {

	It("Initializer Handler, list and ensure expected number of routes", func() {
		mockLog := zap.NewExample().Sugar()
		mockDb, mock, _ := sqlmock.New()
		dialector := postgres.New(postgres.Config{
			Conn:       mockDb,
			DriverName: "postgres",
		})
		db, _ := gorm.Open(dialector, &gorm.Config{})

		// close the go-routine which gets auto-initialized
		ctx, cancel := context.WithCancel(context.Background())
		wg := new(sync.WaitGroup)
		mock.ExpectExec(regexp.QuoteMeta(
			`REFRESH MATERIALIZED VIEW`)).WillReturnResult(sqlmock.NewResult(1, 1))
		h := NewHandler(ctx, wg, mockLog, nil, db)
		cancel()
		wg.Wait()
		gin.SetMode(gin.TestMode)
		router := gin.New()
		apiGroup := router.Group("/api/v1")
		h.RegisterRoutes(apiGroup)
		routes := router.Routes()
		Expect(routes).To(HaveLen(10))
	})
})
