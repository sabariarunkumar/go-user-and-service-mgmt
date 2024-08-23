package integration

import (
	"context"
	"log"
	"sync"
	"testing"
	"userservice/cmd/migration"
	"userservice/internal/components/service"
	"userservice/internal/components/user"
	"userservice/internal/configs"

	"github.com/sabariarunkumar/go-logger"

	"userservice/internal/middleware"
	"userservice/internal/misc"

	"github.com/gin-gonic/gin"
	pginit "github.com/sabariarunkumar/go-postgresql-init"
	gormLogger "gorm.io/gorm/logger"

	"gorm.io/gorm"
)

var db *gorm.DB

func initPrerequisites() error {
	envFile := "runtime.env"
	config, err := configs.InitConfig(envFile)
	if err != nil {
		log.Fatal(err)
	}
	logger := logger.NewLogger(config.LogLevel)

	db, err = pginit.InitDB(
		config.DBHost,
		config.DBPort,
		config.DBUser,
		config.DBName,
		config.DBPassword,
		"mgmt_portal",
		config.DBConnTimeout,
		config.DBMaxConnIdleTime,
		config.DBMaxOpenConn,
		config.DBSlowQueryLogThreshold,
		gormLogger.Silent,
	)
	if err != nil {
		log.Fatal(err)
	}
	err = truncateTables(db)
	if err != nil {
		return err
	}
	err = migration.InitDBEntities(logger, db)
	if err != nil {
		return err
	}
	err = misc.LoadUserRoles(logger, db)
	if err != nil {
		log.Fatal(err)
	}
	misc.InitPayloadValidator()
	gin.SetMode(gin.ReleaseMode)
	router = gin.Default()
	v1Apis := router.Group("/api/v1")
	v1Apis.Use(middleware.Authenticate("/api/v1", logger, []byte(config.JWTSecret)))
	userHandler := user.NewHandler(logger, config, db)
	userHandler.RegisterRoutes(v1Apis)
	ctx := context.Background()
	var wg sync.WaitGroup
	serviceHandler := service.NewHandler(ctx, &wg, logger, config, db)
	serviceHandler.RegisterRoutes(v1Apis)

	return nil
}

func TestIntegrationSuite(t *testing.T) {
	err := initPrerequisites()
	if err != nil {
		t.Errorf("init error %+v", err)
	}
	t.Run("TestAdminLogin", TestAdminLogin)
	t.Run("TestAdminResetPassword", TestAdminResetPassword)
	t.Run("TestAdminListUsers", TestAdminListUsers)
	t.Run("TestAdminAddUser", TestAdminAddUser)
	t.Run("TestConfiguredUserLogin", TestConfiguredUserLogin)
	t.Run("TestConfiguredUserResetPassword", TestConfiguredUserResetPassword)
	t.Run("TestAdvancedUserAddService", TestAdvancedUserAddService)
	t.Run("TestAdvancedUserListServiceVersion", TestAdvancedUserListServiceVersion)
	t.Run("TestAdvancedUserAddsVersionV1", TestAdvancedUserAddsVersionV1)
	t.Run("TestAdvancedUserAddsVersionV2", TestAdvancedUserAddsVersionV2)

	// Refresh views to get updates
	err = db.Exec(`REFRESH MATERIALIZED VIEW name_sorted_service`).Error
	if err != nil {
		t.Errorf("Internal DB error %+v", err)
	}
	t.Run("TestAdvancedUserListService", TestAdvancedUserListService)
	t.Run("TestAdvancedUserListServiceWithSortFilter", TestAdvancedUserListServiceWithSortFilter)

	// cleanups
	_ = truncateTables(db)
}
