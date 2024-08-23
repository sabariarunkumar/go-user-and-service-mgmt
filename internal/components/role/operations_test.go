package role

import (
	"database/sql"
	"errors"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _ = Describe("Fetch Roles [operations]", func() {
	var (
		mockLog *zap.SugaredLogger
		mock    sqlmock.Sqlmock
		mockDb  *sql.DB
		ops     *operations
		db      *gorm.DB
	)
	BeforeEach(func() {
		mockLog = zap.NewExample().Sugar()
		mockDb, mock, _ = sqlmock.New()
		dialector := postgres.New(postgres.Config{
			Conn:       mockDb,
			DriverName: "postgres",
		})
		db, _ = gorm.Open(dialector)
	})

	It("Initialize operations", func() {
		ops = newOperations(db, mockLog)
		Expect(ops).To(Not(BeNil()))
	})
	It("Unexpected DB issues", func() {
		ops = newOperations(db, mockLog)
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("connection is already closed"))
		userRoles, err := ops.FetchRoles()
		Expect(err).To(Not(BeNil()))
		Expect(err.Error()).To(Equal("internal error"))
		Expect(userRoles).To(BeNil())
	})
	It("Successful fetch", func() {
		ops = newOperations(db, mockLog)
		mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"name", "description"}).AddRow("basic", "read"))
		userRoles, err := ops.FetchRoles()
		Expect(err).To(BeNil())
		Expect(userRoles).To(Not(BeNil()))
	})

})
