package misc

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

// func TestUtils(t *testing.T) {
// 	RegisterFailHandler(Fail)
// 	RunSpecs(t, "User Roles Test Suite")
// }

var _ = Describe("Roles Suite Test", func() {
	var (
		mockLog *zap.SugaredLogger
		db      *gorm.DB
		mock    sqlmock.Sqlmock
		mockDb  *sql.DB
	)

	Context("Fetch User Roles", func() {
		BeforeEach(func() {
			mockLog = zap.NewExample().Sugar()
			mockDb, mock, _ = sqlmock.New()
			dialector := postgres.New(postgres.Config{
				Conn:       mockDb,
				DriverName: "postgres",
			})
			db, _ = gorm.Open(dialector)
		})

		It("Fetch configured roles", func() {
			rows := sqlmock.NewRows([]string{"Name", "Description"}).
				AddRow("basic", "<basic-perm>").
				AddRow("admin", "<admin-perm>").
				AddRow("advanced", "<advanced-perm>")
			mock.ExpectQuery(`SELECT`).WillReturnRows(rows)
			err := LoadUserRoles(mockLog, db)
			Expect(err).To(BeNil())
			Expect(Roles).To(HaveKeyWithValue("basic", "<basic-perm>"))
			Expect(Roles).To(HaveKeyWithValue("admin", "<admin-perm>"))
			Expect(Roles).To(HaveKeyWithValue("advanced", "<advanced-perm>"))
			Expect(mock.ExpectationsWereMet()).To(BeNil())
		})
		It("Behavior of zero rows fetch", func() {
			rows := sqlmock.NewRows([]string{"Name", "Description"})
			mock.ExpectQuery(`SELECT`).WillReturnRows(rows)
			err := LoadUserRoles(mockLog, db)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("no User roles configured; Run Migration"))
			Expect(mock.ExpectationsWereMet()).To(BeNil())
		})
		It("Unexpected DB issues", func() {
			mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("connection is already closed"))
			err := LoadUserRoles(mockLog, db)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("unable to fetch user roles from DB: connection is already closed"))
			Expect(mock.ExpectationsWereMet()).To(BeNil())
		})
	})
})
