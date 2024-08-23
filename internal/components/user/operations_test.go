package user

import (
	"database/sql"
	"errors"
	"regexp"
	"time"
	appErrors "userservice/internal/errors"
	"userservice/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _ = Describe("Users [operations]", func() {
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
		ops = newOperations(db, mockLog)
	})

	It("Initialize operations", func() {
		Expect(ops).To(Not(BeNil()))
	})
	Context("get user by email", func() {
		It("No user with email", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user"`)).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "email", "role",
						"password_hash", "temp_password"}))

			user, err := ops.GetUserByEmail("admin@mgmtportal.com")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(gorm.ErrRecordNotFound))
			Expect(user).To(BeNil())
		})
		It("Internal DB error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user"`)).
				WillReturnError(errors.New("connection is already closed"))

			user, err := ops.GetUserByEmail("admin@mgmtportal.com")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(user).To(BeNil())
		})
		It("user with expected email", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user"`)).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "email", "role",
						"password_hash", "temp_password"}).AddRow("1",
						time.Time{},
						time.Time{},
						time.Time{},
						"admin",
						"admin@mgmtportal.com",
						"admin",
						"xyz",
						true))

			user, err := ops.GetUserByEmail("admin@mgmtportal.com")
			Expect(err).To(BeNil())
			Expect(user).To(Not(BeNil()))
			Expect(user.Email).To(Equal("admin@mgmtportal.com"))
		})
	})
	Context("get user by ID", func() {
		It("No user with id", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user"`)).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "email", "role",
						"password_hash", "temp_password"}))

			user, err := ops.GetUser(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(gorm.ErrRecordNotFound))
			Expect(user).To(BeNil())
		})
		It("Internal DB error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user"`)).
				WillReturnError(errors.New("connection is already closed"))

			user, err := ops.GetUser(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(user).To(BeNil())
		})
		It("fetching Valid user", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user"`)).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "email", "role",
						"password_hash", "temp_password"}).AddRow("1",
						time.Time{},
						time.Time{},
						time.Time{},
						"admin",
						"admin@mgmtportal.com",
						"admin",
						"xyz",
						true))

			user, err := ops.GetUser(1)
			Expect(err).To(BeNil())
			Expect(user).To(Not(BeNil()))
			Expect(user.ID).To(Equal(uint(1)))
		})
	})
	Context("create/insert user record", func() {
		It("Internal DB error while checking for user with same email", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnError(errors.New("connection is already closed"))

			err := ops.CreateUser("admin", "admin@mgmtportal.com", "basic", "hash")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("user already exist with desired email", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			err := ops.CreateUser("admin", "admin@mgmtportal.com", "basic", "hash")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrUserWithSameEmailAlreadyExists))
		})
		It("In distributed/concurrent env, while proceeding to create email,"+
			" we experience UniqueKey Constrain Violation due to same email", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(
				`INSERT INTO "user"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					"admin",
					"admin@mgmtportal.com",
					"basic",
					"hash",
					true,
				).WillReturnError(appErrors.ErrUniqueKeyConstrainViolation)
			mock.ExpectRollback()
			err := ops.CreateUser("admin", "admin@mgmtportal.com", "basic", "hash")
			Expect(err).To(MatchError(appErrors.ErrUserWithSameEmailAlreadyExists))
		})
		It("In distributed/concurrent env, while proceeding to create email, we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(
				`INSERT INTO "user"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					"admin",
					"admin@mgmtportal.com",
					"basic",
					"hash",
					true,
				).WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			err := ops.CreateUser("admin", "admin@mgmtportal.com", "basic", "hash")
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("successfully create user", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(
				`INSERT INTO "user"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					"admin",
					"admin@mgmtportal.com",
					"basic",
					"hash",
					true,
				).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))
			mock.ExpectCommit()
			err := ops.CreateUser("admin", "admin@mgmtportal.com", "basic", "hash")
			Expect(err).To(BeNil())
		})
	})
	Context("Update user record by ID", func() {
		It("No user with ID", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}))
			user, err := ops.UpdateUser(1, "admin", "admin@mgmtportal.com", "basic")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrUserDoesNotExist))
			Expect(user).To(BeNil())
		})
		It("Internal error while determining user exists with ID", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnError(errors.New("connection error"))
			user, err := ops.UpdateUser(1, "admin", "admin@mgmtportal.com", "basic")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(user).To(BeNil())
		})
		It("assuming there is already a record in the system that"+
			"contains the email address specified in an update request ", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE (email = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			user, err := ops.UpdateUser(1, "admin", "admin@mgmtportal.com", "basic")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrUserWithSameEmailAlreadyExists))
			Expect(user).To(BeNil())
		})
		It("Connection error while verifying a record in the system that contains"+
			"the email address specified in an update request", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE (email = $1 and id != $2)`)).
				WillReturnError(errors.New("connection error"))
			user, err := ops.UpdateUser(1, "admin", "admin@mgmtportal.com", "basic")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(user).To(BeNil())
		})

		It("In distributed/concurrent env, while proceeding to update email,"+
			"we experience UniqueKey Constrain Violation due to same email existence", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE (email = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "user" SET "id"=$1`)).
				WillReturnError(appErrors.ErrUniqueKeyConstrainViolation)
			mock.ExpectRollback()
			user, err := ops.UpdateUser(1, "admin", "adminv2@mgmtportal.com", "basic")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrUserWithSameEmailAlreadyExists))
			Expect(user).To(BeNil())
		})
		It("In distributed/concurrent env, while proceeding to update email, we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE (email = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "user" SET "id"=$1`)).
				WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			user, err := ops.UpdateUser(1, "admin", "adminv2@mgmtportal.com", "basic")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(user).To(BeNil())
		})
		It("In distributed/concurrent env, while proceeding to update a record, we experience record not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE (email = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "user" SET "id"=$1`)).
				WillReturnError(gorm.ErrRecordNotFound)
			mock.ExpectRollback()
			user, err := ops.UpdateUser(1, "admin", "adminv2@mgmtportal.com", "basic")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrUserDoesNotExist))
			Expect(user).To(BeNil())
		})
		It("successful update request", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE (email = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "user" SET "id"=$1`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			user, err := ops.UpdateUser(1, "admin", "adminv2@mgmtportal.com", "basic")
			Expect(err).To(BeNil())
			Expect(user.Email).To(Equal("adminv2@mgmtportal.com"))
		})
	})
	Context("Delete user record by ID", func() {
		It("No user with ID", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}))
			err := ops.DeleteUser(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrUserDoesNotExist))
		})
		It("Internal error while determining user exists with ID", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnError(errors.New("connection error"))
			err := ops.DeleteUser(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("In distributed/concurrent env, while proceeding to delete record we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "user" WHERE "user"."id" = $1`)).
				WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			err := ops.DeleteUser(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("successful delete request", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "user" WHERE "user"."id" = $1`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			err := ops.DeleteUser(1)
			Expect(err).To(BeNil())
		})
	})
	Context("Fetch user records with page", func() {
		It("Internal error while getting total count", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).WillReturnError(errors.New("connection error"))
			users, total, err := ops.FetchUsersWithPagination(1, 10)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(users).To(HaveLen(0))
			Expect(total).To(Equal(int64(0)))

		})
		It("Internal error while getting matching users", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE "user"."deleted_at" IS NULL LIMIT $1`)).
				WillReturnError(errors.New("connection error"))
			users, total, err := ops.FetchUsersWithPagination(1, 10)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(users).To(HaveLen(0))
			Expect(total).To(Equal(int64(0)))
		})
		It("Successful fetch", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE "user"."deleted_at" IS NULL LIMIT $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "email", "role",
					"password_hash", "temp_password"}).AddRow("1",
					time.Time{},
					time.Time{},
					time.Time{},
					"admin",
					"admin@mgmtportal.com",
					"admin",
					"xyz",
					true))
			users, total, err := ops.FetchUsersWithPagination(0, 1)
			Expect(err).To(BeNil())
			Expect(users).To(HaveLen(1))
			Expect(total).To(Equal(int64(1)))

		})
	})
	Context("Format user records with page", func() {
		res := ops.FormatUserDetailsWithPageDetails([]models.User{{Email: "mgmtportal@gmail.com"}}, 1, 1, 1)
		Expect(res.TotalItems).To(Equal(int64(1)))
	})
	Context("Change Password", func() {
		It("Internal error while determining user exists for given email", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnError(errors.New("connection error"))
			err := ops.ChangePassword("mgmtportal@admin.com", "hash")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("User doesn't exists for given email", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			err := ops.ChangePassword("mgmtportal@admin.com", "hash")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrUserDoesNotExist))
		})
		It("In distributed/concurrent env, while proceeding to update password, we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "user" SET`)).WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			err := ops.ChangePassword("mgmtportal@admin.com", "hash")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("In distributed/concurrent env, while proceeding to update password, we experience record not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "user" SET`)).WillReturnError(gorm.ErrRecordNotFound)
			mock.ExpectRollback()
			err := ops.ChangePassword("mgmtportal@admin.com", "hash")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrUserDoesNotExist))
		})
		It("successful update request", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE email = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "user" SET`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			err := ops.ChangePassword("mgmtportal@admin.com", "hash")
			Expect(err).To(BeNil())
		})
	})
})
