package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sync"
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

var _ = Describe("Services [operations]", func() {
	var (
		mockLog *zap.SugaredLogger
		mock    sqlmock.Sqlmock
		mockDb  *sql.DB
		ops     *operations
		db      *gorm.DB
		ctxBg   context.Context
		wg      *sync.WaitGroup
	)
	BeforeEach(func() {
		mockLog = zap.NewExample().Sugar()
		mockDb, mock, _ = sqlmock.New()
		dialector := postgres.New(postgres.Config{
			Conn:       mockDb,
			DriverName: "postgres",
		})
		db, _ = gorm.Open(dialector)
		ctxBg = context.Background()
		wg = new(sync.WaitGroup)
		ops = &operations{db: db, log: mockLog, ctx: ctxBg}
		_ = mock
	})
	It("Initialize operations", func() {
		Expect(ops).To(Not(BeNil()))
	})
	Context("SyncDataWithSortedViews", func() {
		var ctx context.Context
		var cancel context.CancelFunc
		BeforeEach(func() {
			ctx, cancel = context.WithCancel(ctxBg)
		})

		It("Context done [simulating main goroutine closing context]", func() {
			ops.ctx = ctx
			// Make first refresh attempt fail and 2nd attempt pass
			mock.ExpectExec(regexp.QuoteMeta(
				`REFRESH MATERIALIZED VIEW`)).WillReturnError(appErrors.ErrInternal)
			mock.ExpectExec(regexp.QuoteMeta(
				`REFRESH MATERIALIZED VIEW`)).WillReturnResult(sqlmock.NewResult(1, 1))
			wg.Add(1)
			go func() {
				defer wg.Done()
				ops.SyncDataWithSortedViews()
			}()
			cancel()
			wg.Wait()
			if err := mock.ExpectationsWereMet(); err != nil {
				Fail("there were unfulfilled expectations while refreshing views")
			}
		})
		It("Ensure during RefreshRequestCheckTicker refresh is handled and state is set", func() {
			ops.ctx = ctx
			// Make initial refresh attempt pass
			mock.ExpectExec(regexp.QuoteMeta(
				`REFRESH MATERIALIZED VIEW`)).WillReturnResult(sqlmock.NewResult(1, 1))
			// Satisfy expectation in first attempt of ViewsScheduledSyncTimeTicker
			mock.ExpectExec(regexp.QuoteMeta(
				`REFRESH MATERIALIZED VIEW`)).WillReturnResult(sqlmock.NewResult(1, 1))
			ops.toRefreshViews = true
			wg.Add(1)
			go func() {
				defer wg.Done()
				models.ViewRefreshRequestCheckTime = 1 * time.Second
				ops.SyncDataWithSortedViews()
			}()
			for {
				time.Sleep(1 * time.Second)
				var toRefresh bool
				ops.mux.RLock()
				toRefresh = ops.toRefreshViews
				ops.mux.RUnlock()
				if !toRefresh {
					break
				}
			}
			cancel()
			wg.Wait()
			if err := mock.ExpectationsWereMet(); err != nil {
				Fail("there were unfulfilled expectations while refreshing views")
			}
		})
	})
	Context("CheckIfServiceExist", func() {
		It("No service with id", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}))

			exists, returnErr := ops.CheckIfServiceExist(1)
			Expect(returnErr).To(BeNil())
			Expect(exists).To(BeFalse())
		})
		It("Internal error[connection closed]", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnError(errors.New("connection is already closed"))

			exists, returnErr := ops.CheckIfServiceExist(1)
			Expect(returnErr).To(Not(BeNil()))
			Expect(returnErr).To(MatchError(appErrors.ErrInternal))
			Expect(exists).To(BeFalse())
		})
		It("service exists", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			exists, returnErr := ops.CheckIfServiceExist(1)
			Expect(returnErr).To(BeNil())
			Expect(exists).To(BeTrue())
		})
	})
	Context("CheckIfServiceVersionExist", func() {
		It("No service with id", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}))

			exists, returnErr := ops.CheckIfVersionForServiceExist(1, "v1")
			Expect(returnErr).To(BeNil())
			Expect(exists).To(BeFalse())
		})
		It("Internal error[connection closed]", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnError(errors.New("connection is already closed"))
			exists, returnErr := ops.CheckIfVersionForServiceExist(1, "v1")
			Expect(returnErr).To(Not(BeNil()))
			Expect(returnErr).To(MatchError(appErrors.ErrInternal))
			Expect(exists).To(BeFalse())
		})
		It("service version exists", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			exists, returnErr := ops.CheckIfVersionForServiceExist(1, "v1")
			Expect(returnErr).To(BeNil())
			Expect(exists).To(BeTrue())
		})
	})
	Context("get service by ID", func() {
		It("No service with id", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "version_count"}))

			service, err := ops.GetService(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceDoesNotExist))
			Expect(service).To(BeNil())
		})
		It("fetching valid service", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "version_count"}).AddRow("1",
					time.Time{},
					time.Time{},
					time.Time{},
					"postman",
					"Nice Product",
					1))

			service, err := ops.GetService(1)
			Expect(err).To(BeNil())
			Expect(service).To(Not(BeNil()))
			Expect(service.ID).To(Equal(uint(1)))
		})
		It("Internal error[connection closed]", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "service" WHERE id = $1`)).
				WillReturnError(errors.New("connection is already closed"))

			service, err := ops.GetService(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(service).To(BeNil())
		})
	})
	Context("create/insert service record", func() {
		It("Internal error[connection closed] while checking for record with same service name", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE name = $1`)).
				WillReturnError(errors.New("connection is already closed"))

			service, err := ops.CreateService("postman", "Nice Product")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(service).To(BeNil())
		})
		It("service already exist with same name", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE name = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			service, err := ops.CreateService("postman", "Nice Product")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceAlreadyExists))
			Expect(service).To(BeNil())

		})
		It("In distributed/concurrent env, while proceeding to create service,"+
			"we experience UniqueKey Constrain Violation due to same name", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE name = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(
				`INSERT INTO "service"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					"postman",
					"Nice Product",
					0,
				).WillReturnError(appErrors.ErrUniqueKeyConstrainViolation)
			mock.ExpectRollback()
			service, err := ops.CreateService("postman", "Nice Product")
			Expect(err).To(MatchError(appErrors.ErrServiceAlreadyExists))
			Expect(service).To(BeNil())
		})
		It("In distributed/concurrent env, while proceeding to create service,"+
			"we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE name = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(
				`INSERT INTO "service"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					"postman",
					"Nice Product",
					0,
				).WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			service, err := ops.CreateService("postman", "Nice Product")
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(service).To(BeNil())
		})
		It("successfully create service", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE name = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(
				`INSERT INTO "service"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					"postman",
					"Nice Product",
					0,
				).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))
			mock.ExpectCommit()
			service, err := ops.CreateService("postman", "Nice Product")
			Expect(err).To(BeNil())
			Expect(service.Name).To(Equal("postman"))
		})
	})
	Context("Update service record by ID", func() {
		It("No service with ID", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}))
			service, err := ops.UpdateService(1, "postman", "Nice Product")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceDoesNotExist))
			Expect(service).To(BeNil())
		})
		It("Internal error while determining service exists with ID", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnError(errors.New("connection error"))
			service, err := ops.UpdateService(1, "postman", "Nice Product")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(service).To(BeNil())
		})
		It("assume there is already a record in the system that contains the same service name"+
			"specified in an update request ", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE (name = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			service, err := ops.UpdateService(1, "postman", "Nice Product")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceAlreadyExists))
			Expect(service).To(BeNil())
		})
		It("Connection error while verifying a record in the system that contains the service"+
			"name specified in an update request ", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE (name = $1 and id != $2)`)).
				WillReturnError(errors.New("connection error"))
			service, err := ops.UpdateService(1, "postman", "Nice Product")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(service).To(BeNil())
		})

		It("In distributed/concurrent env, while proceeding to update service,"+
			"we experience UniqueKey Constrain Violation due to same serviceName existence", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE (name = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "service" SET "id"=$1`)).
				WillReturnError(appErrors.ErrUniqueKeyConstrainViolation)
			mock.ExpectRollback()
			service, err := ops.UpdateService(1, "postman", "Nice Product")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceAlreadyExists))
			Expect(service).To(BeNil())
		})
		It("In distributed/concurrent env, while proceeding to update service, we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE (name = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "service" SET "id"=$1`)).
				WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			service, err := ops.UpdateService(1, "postman", "Nice Product")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(service).To(BeNil())
		})
		It("In distributed/concurrent env, while proceeding to update a record, we experience record not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE (name = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "service" SET "id"=$1`)).
				WillReturnError(gorm.ErrRecordNotFound)
			mock.ExpectRollback()
			service, err := ops.UpdateService(1, "postman", "Nice Product")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceDoesNotExist))
			Expect(service).To(BeNil())
		})
		It("successful update request", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE (name = $1 and id != $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "service" SET "id"=$1`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			service, err := ops.UpdateService(1, "postman", "Nice Product")
			Expect(err).To(BeNil())
			Expect(service.Name).To(Equal("postman"))
		})
	})
	Context("Delete service record by ID", func() {
		It("No service with ID", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}))
			err := ops.DeleteService(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceDoesNotExist))
		})
		It("Internal error while determining service exists with ID", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnError(errors.New("connection error"))
			err := ops.DeleteService(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("In distributed/concurrent env, while proceeding to delete service version(s),"+
			"we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "version" WHERE service_id = $1`)).
				WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			err := ops.DeleteService(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("In distributed/concurrent env, while proceeding to delete service, we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "version" WHERE service_id = $1`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "service" WHERE "service"."id" = $1`)).
				WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			err := ops.DeleteService(1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("successful delete request", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "version" WHERE service_id = $1`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "service" WHERE "service"."id" = $1`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			err := ops.DeleteService(1)
			Expect(err).To(BeNil())
		})
	})
	Context("Fetch services", func() {
		searchStr := fmt.Sprintf("%%%s%%", "service")
		It("Internal error while getting total count of date sorted service", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE name like`)).
				WillReturnError(errors.New("connection error"))
			services, total, err := ops.FetchServices(1, 1, searchStr, false, false)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(services).To(HaveLen(0))
			Expect(total).To(Equal(int64(0)))

		})
		It("Internal error while getting matching services", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service" WHERE name like`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(
				regexp.QuoteMeta(`SELECT * FROM "service" WHERE name like $1 AND "service"."deleted_at" IS NULL LIMIT $2`)).
				WillReturnError(errors.New("connection error"))
			services, total, err := ops.FetchServices(1, 1, searchStr, false, false)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(services).To(HaveLen(0))
			Expect(total).To(Equal(int64(0)))
		})
		It("Successful fetch", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service"`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "service" WHERE name like $1 AND "service"."deleted_at" IS NULL LIMIT $2`)).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "version_count"}).
						AddRow("1",
							time.Time{},
							time.Time{},
							time.Time{},
							"admin",
							"admin@mgmtportal.com",
							1))
			services, total, err := ops.FetchServices(1, 1, searchStr, false, false)
			Expect(err).To(BeNil())
			Expect(services).To(HaveLen(1))
			Expect(total).To(Equal(int64(1)))

		})
		It("Successful Inverted fetch", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service"`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			mock.ExpectQuery(
				regexp.QuoteMeta(`SELECT * FROM "service" WHERE name like $1 AND "service"."deleted_at" IS NULL LIMIT $2`)).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "version_count"}).
						AddRow("1",
							time.Time{},
							time.Time{},
							time.Time{},
							"admin",
							"admin@mgmtportal.com",
							1).AddRow("2",
						time.Time{},
						time.Time{},
						time.Time{},
						"admin",
						"admin@mgmtportal.com",
						1))
			services, total, err := ops.FetchServices(3, 1, searchStr, true, false)
			Expect(err).To(BeNil())
			Expect(services).To(HaveLen(2))
			Expect(services[0].ID).To(Equal(uint(2)))
			Expect(total).To(Equal(int64(2)))

		})
		It("Successful Inverted fetch [Non existing page]", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "service"`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			mock.ExpectQuery(
				regexp.QuoteMeta(`SELECT * FROM "service" WHERE name like $1 AND "service"."deleted_at" IS NULL LIMIT $2`)).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "version_count"}).
						AddRow("1",
							time.Time{},
							time.Time{},
							time.Time{},
							"admin",
							"admin@mgmtportal.com",
							1).AddRow("2",
						time.Time{},
						time.Time{},
						time.Time{},
						"admin",
						"admin@mgmtportal.com",
						1))
			services, total, err := ops.FetchServices(4, 1, searchStr, true, false)
			Expect(err).To(BeNil())
			Expect(services).To(HaveLen(0))
			Expect(total).To(Equal(int64(2)))

		})
		It("Successful NameSorted Service fetch", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "name_sorted_service"`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "name_sorted_service" WHERE name like $1`)).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "version_count"}).
						AddRow("1",
							time.Time{},
							time.Time{},
							time.Time{},
							"admin",
							"admin@mgmtportal.com",
							1).AddRow("2",
						time.Time{},
						time.Time{},
						time.Time{},
						"admin",
						"admin@mgmtportal.com",
						1))
			services, total, err := ops.FetchServices(1, 1, searchStr, false, true)
			Expect(err).To(BeNil())
			Expect(services).To(HaveLen(2))
			Expect(services[0].ID).To(Equal(uint(1)))
			Expect(total).To(Equal(int64(2)))

		})
	})
	Context("Format service records with page", func() {
		res := ops.FormatServiceDetailsWithPageDetails([]models.Service{{Name: "postman"}}, 1, 1, 1)
		Expect(res.TotalItems).To(Equal(int64(1)))
	})
	Context("get service version by ID and version tag", func() {
		It("No service version with id and version tag", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "tag", "info"}))

			serviceVersion, err := ops.GetServiceVersion(1, "v1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceVersionDoesNotExist))
			Expect(serviceVersion).To(BeNil())
		})
		It("Internal error[connection closed]", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnError(errors.New("connection error"))

			serviceVersion, err := ops.GetServiceVersion(1, "v1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(serviceVersion).To(BeNil())
		})
		It("fetching service by valid id and version tag ", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "tag", "info"}).AddRow("1",
					time.Time{},
					time.Time{},
					time.Time{},
					"v1",
					"version-1",
				))

			serviceVersion, err := ops.GetServiceVersion(1, "v1")
			Expect(err).To(BeNil())
			Expect(serviceVersion).To(Not(BeNil()))
			Expect(serviceVersion.Tag).To(Equal("v1"))
		})

	})
	Context("create/insert service record", func() {
		It("Internal error[connection closed] while checking for record with same service name", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnError(errors.New("connection is already closed"))

			serviceVersion, err := ops.CreateServiceVersion(1, "v1", "version-1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(serviceVersion).To(BeNil())
		})
		It("service already exist with same name ", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			serviceVersion, err := ops.CreateServiceVersion(1, "v1", "version-1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceVersionAlreadyExists))
			Expect(serviceVersion).To(BeNil())

		})
		It("In distributed/concurrent env, while proceeding to create service version,"+
			"we experience UniqueKey Constrain Violation due to same name", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`INSERT INTO "version"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					uint(1),
					"v1",
					"version-1",
				).WillReturnError(appErrors.ErrUniqueKeyConstrainViolation)
			mock.ExpectRollback()
			serviceVersion, err := ops.CreateServiceVersion(1, "v1", "version-1")
			Expect(err).To(MatchError(appErrors.ErrServiceVersionAlreadyExists))
			Expect(serviceVersion).To(BeNil())
		})
		It("In distributed/concurrent env, while proceeding to create service version, we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`INSERT INTO "version"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					uint(1),
					"v1",
					"version-1",
				).WillReturnError(appErrors.ErrInternal)
			mock.ExpectRollback()
			serviceVersion, err := ops.CreateServiceVersion(1, "v1", "version-1")
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(serviceVersion).To(BeNil())
		})
		It("In distributed/concurrent env, while proceeding to update version_count"+
			" in Service table, we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`INSERT INTO "version"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					uint(1),
					"v1",
					"version-1",
				).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(regexp.QuoteMeta(`UPDATE "service" SET "version_count"=version_count + $1`)).
				WillReturnError(appErrors.ErrInternal)
			mock.ExpectRollback()

			serviceVersion, err := ops.CreateServiceVersion(1, "v1", "version-1")
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(serviceVersion).To(BeNil())
		})
		It("successfully creation of service version", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`INSERT INTO "version"`)).
				WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					uint(1),
					"v1",
					"version-1",
				).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(regexp.QuoteMeta(`UPDATE "service" SET "version_count"=version_count + $1`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			serviceVersion, err := ops.CreateServiceVersion(1, "v1", "version-1")
			Expect(err).To(BeNil())
			Expect(serviceVersion.Tag).To(Equal("v1"))
		})
	})
	Context("Update service record by ID", func() {
		It("Internal error[connection closed] while checking for record with same version tag", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnError(errors.New("connection is already closed"))

			serviceVersion, err := ops.UpdateServiceVersion(1, "v1", "version-1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(serviceVersion).To(BeNil())
		})
		It("service version doesn't exist ", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}))
			serviceVersion, err := ops.UpdateServiceVersion(1, "v1", "version-1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceVersionDoesNotExist))
			Expect(serviceVersion).To(BeNil())
		})
		It("In distributed/concurrent env, while proceeding to update service version, we experience record not found",
			func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "version" SET`)).
					WillReturnError(gorm.ErrRecordNotFound)
				mock.ExpectRollback()
				serviceVersion, err := ops.UpdateServiceVersion(1, "v1", "version-1")
				Expect(err).To(Not(BeNil()))
				Expect(err).To(MatchError(appErrors.ErrServiceVersionDoesNotExist))
				Expect(serviceVersion).To(BeNil())
			})
		It("In distributed/concurrent env, while proceeding to update service version, we experience Internal Error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "version" SET`)).
				WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			serviceVersion, err := ops.UpdateServiceVersion(1, "v1", "version-1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(serviceVersion).To(BeNil())
		})
		It("successful update request", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "version" SET`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			serviceVersion, err := ops.UpdateServiceVersion(1, "v1", "version-1")
			Expect(err).To(BeNil())
			Expect(serviceVersion.Tag).To(Equal("v1"))
		})
	})
	Context("Delete service version record by ID", func() {
		It("Internal error[connection closed] while checking for record with same version tag", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnError(errors.New("connection is already closed"))

			err := ops.DeleteServiceVersion(1, "v1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("service version doesn't exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}))
			err := ops.DeleteServiceVersion(1, "v1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceVersionDoesNotExist))
		})
		It("In distributed/concurrent env, while proceeding to delete service version,"+
			" we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "version" WHERE tag = $1 and service_id = $2`)).
				WillReturnError(errors.New("connection error"))
			mock.ExpectRollback()
			err := ops.DeleteServiceVersion(1, "v1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("In distributed/concurrent env, while proceeding to delete service version,"+
			" we experience record not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "version" WHERE tag = $1 and service_id = $2`)).
				WillReturnError(gorm.ErrRecordNotFound)
			mock.ExpectRollback()
			err := ops.DeleteServiceVersion(1, "v1")
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrServiceVersionDoesNotExist))
		})
		It("In distributed/concurrent env, while proceeding to update version_count in Service table,"+
			"we experience Internal error", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "version" WHERE tag = $1 and service_id = $2`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(regexp.QuoteMeta(`UPDATE "service" SET "version_count"=version_count - $1`)).
				WillReturnError(appErrors.ErrInternal)
			mock.ExpectRollback()

			err := ops.DeleteServiceVersion(1, "v1")
			Expect(err).To(MatchError(appErrors.ErrInternal))
		})
		It("successfully deletion of service version", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE (tag = $1 and service_id = $2)`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`DELETE FROM "version" WHERE tag = $1 and service_id = $2`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(regexp.QuoteMeta(`UPDATE "service" SET "version_count"=version_count - $1`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			err := ops.DeleteServiceVersion(1, "v1")
			Expect(err).To(BeNil())
		})
	})
	Context("Fetch service version", func() {
		It("Internal error while getting total count of date sorted service", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE service_id = $1`)).
				WillReturnError(errors.New("connection error"))
			serviceVersions, total, err := ops.FetchServiceVersionsInverted(1, 1, 1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(serviceVersions).To(HaveLen(0))
			Expect(total).To(Equal(int64(0)))
		})
		It("Internal error while getting matching service", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE service_id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(
				regexp.QuoteMeta(`SELECT * FROM "version" WHERE service_id = $1 AND "version"."deleted_at" IS NULL LIMIT $2`)).
				WillReturnError(errors.New("connection error"))
			serviceVersions, total, err := ops.FetchServiceVersionsInverted(1, 2, 1)
			Expect(err).To(Not(BeNil()))
			Expect(err).To(MatchError(appErrors.ErrInternal))
			Expect(serviceVersions).To(HaveLen(0))
			Expect(total).To(Equal(int64(0)))
		})
		It("Successful fetch", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE service_id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			mock.ExpectQuery(
				regexp.QuoteMeta(`SELECT * FROM "version" WHERE service_id = $1 AND "version"."deleted_at" IS NULL LIMIT $2`)).
				WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at", "deleted_at", "service_id", "tag", "info"}).
					AddRow(
						time.Time{},
						time.Time{},
						time.Time{},
						1,
						"v1",
						"version-1").AddRow(
					time.Time{},
					time.Time{},
					time.Time{},
					1,
					"v1",
					"version-1"))
			serviceVersions, total, err := ops.FetchServiceVersionsInverted(1, 1, 3)
			Expect(err).To(BeNil())
			Expect(serviceVersions).To(HaveLen(2))
			Expect(total).To(Equal(int64(2)))

		})
		It("Successful fetch [non-existing page]", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "version" WHERE service_id = $1`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			mock.ExpectQuery(
				regexp.QuoteMeta(`SELECT * FROM "version" WHERE service_id = $1 AND "version"."deleted_at" IS NULL LIMIT $2`)).
				WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at", "deleted_at", "service_id", "tag", "info"}).
					AddRow(
						time.Time{},
						time.Time{},
						time.Time{},
						1,
						"v1",
						"version-1").AddRow(
					time.Time{},
					time.Time{},
					time.Time{},
					1,
					"v1",
					"version-1"))
			serviceVersions, total, err := ops.FetchServiceVersionsInverted(1, 6, 3)
			Expect(err).To(BeNil())
			Expect(serviceVersions).To(HaveLen(0))
			Expect(total).To(Equal(int64(2)))

		})
	})
	Context("Format service version records", func() {
		res := ops.FormatVersionDetailsWithPageDetails([]models.ServiceVersion{{Tag: "v1"}}, 1, 1, 1)
		Expect(res.TotalItems).To(Equal(int64(1)))
	})
})
