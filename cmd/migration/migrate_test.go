package migration

import (
	"database/sql"
	"errors"
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _ = Describe("Initialize DB entities", func() {
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
		It("DB Connection Error while checking for email existence/count", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).WillReturnError(errors.New("connection is already closed"))
			err := InitDBEntities(mockLog, db)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("internal error while adding admin user: connection is already closed"))
		})
		It("DB Connection Error while creating email", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).WillReturnRows(sqlmock.NewRows([]string{"Count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(
				`INSERT INTO "user"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					"admin",
					"admin@mgmtportal.com",
					"admin",
					sqlmock.AnyArg(),
					true,
				).WillReturnError(errors.New("connection is already closed"))
			mock.ExpectRollback()
			err := InitDBEntities(mockLog, db)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("failed to create admin user: connection is already closed"))
		})

		It("DB Connection Error while checking of basic user role", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).WillReturnRows(sqlmock.NewRows([]string{"Count"}).AddRow(0))
			mock.ExpectBegin()

			mock.ExpectQuery(regexp.QuoteMeta(
				`INSERT INTO "user"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					"admin",
					"admin@mgmtportal.com",
					"admin",
					sqlmock.AnyArg(),
					true,
				).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))
			mock.ExpectCommit()

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user_role"`)).WillReturnError(errors.New("connection is already closed"))
			err := InitDBEntities(mockLog, db)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("connection is already closed"))
		})
	})

	It("DB Connection Error while creating basic user role", func() {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).WillReturnRows(sqlmock.NewRows([]string{"Count"}).AddRow(0))
		mock.ExpectBegin()

		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "user"`)).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				"admin",
				"admin@mgmtportal.com",
				"admin",
				sqlmock.AnyArg(),
				true,
			).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))
		mock.ExpectCommit()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user_role"`)).WillReturnRows(sqlmock.NewRows([]string{"Count"}).AddRow(0))

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`INSERT INTO "user_role"`)).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).WillReturnError(errors.New("connection is already closed"))
		mock.ExpectRollback()

		err := InitDBEntities(mockLog, db)
		Expect(err).To(Not(BeNil()))
		Expect(err.Error()).To(ContainSubstring("connection is already closed"))
	})
	It("Successful Initialization of all items", func() {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).WillReturnRows(sqlmock.NewRows([]string{"Count"}).AddRow(0))
		mock.ExpectBegin()

		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "user"`)).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				"admin",
				"admin@mgmtportal.com",
				"admin",
				sqlmock.AnyArg(),
				true,
			).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))
		mock.ExpectCommit()
		var err error
		for _, role := range defaultRoles() {
			_ = role
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user_role"`)).WillReturnRows(sqlmock.NewRows([]string{"Count"}).AddRow(0))

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`INSERT INTO "user_role"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
				).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
		}
		err = InitDBEntities(mockLog, db)
		Expect(err).To(BeNil())
	})

})
