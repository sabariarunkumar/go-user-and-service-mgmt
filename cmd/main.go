package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"userservice/cmd/api"
	"userservice/cmd/migration"
	"userservice/internal/configs"
	"userservice/internal/misc"

	"github.com/sabariarunkumar/go-logger"

	pginit "github.com/sabariarunkumar/go-postgresql-init"
	gormLogger "gorm.io/gorm/logger"
)

const (
	appNameInDatabase   = "mgmt_portal"
	defaultLogLevelGORM = "debug"
)

func main() {
	dbMigration := flag.Bool(
		"migrate",
		false,
		"migrate DB entities and exit",
	)
	envFile := flag.String(
		"env-file",
		"",
		"migrate DB entities and exit",
	)
	flag.Parse()

	// Runtime config.
	config, err := configs.InitConfig(*envFile)
	if err != nil {
		log.Fatal(err)
	}
	// Custom logger used by application.
	logger := logger.NewLogger(config.LogLevel)

	// let us set gorm log level to silent by default
	// In case of app logger set to DEBUG,
	// we shall set gorm accepted detailed Logging level `INFO`
	gormLogLevel := gormLogger.Silent
	if strings.ToLower(config.LogLevel) == defaultLogLevelGORM {
		gormLogLevel = gormLogger.Info
	}

	db, err := pginit.InitDB(
		config.DBHost,
		config.DBPort,
		config.DBUser,
		config.DBName,
		config.DBPassword,
		appNameInDatabase,
		config.DBConnTimeout,
		config.DBMaxConnIdleTime,
		config.DBMaxOpenConn,
		config.DBSlowQueryLogThreshold,
		gormLogLevel,
	)
	if err != nil {
		logger.Fatal(err)
	}

	if *dbMigration {
		err := migration.MigrateDBEntities(logger, db)
		if err != nil {
			logger.Fatal(err)
		}
		err = migration.InitDBEntities(logger, db)
		if err != nil {
			logger.Fatal(err)
		}
		os.Exit(0)
	}

	err = misc.LoadUserRoles(logger, db)
	if err != nil {
		logger.Fatal(err)
	}
	misc.InitPayloadValidator()

	var (
		// runtimeContext controls the runtime of goroutine which periodically refresh db materialized view.
		runtimeContext, runtimeContextCancel = context.WithCancel(context.Background())
		// waitgroup ensures if the db view refresh goroutine and api service is stopped.
		wg sync.WaitGroup
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	apiServerError := make(chan error, 1)

	apiServer := api.NewAPIServer(runtimeContext, &wg, config, db, logger)
	serverRuntime := apiServer.StartAPIServer(&wg, apiServerError)

	select {
	case err := <-apiServerError:
		logger.Errorf("Failed to run api server: %+v", err)
		runtimeContextCancel()
	case <-stop:
		logger.Info("Received an OS signal, shutting down gracefully...")
		runtimeContextCancel()
		// Api server graceful shutdown
		if err := serverRuntime.Shutdown(runtimeContext); err != nil {
			logger.Errorf("Failed to shutdown api server gracefully: %+v", err)
		}
	}
	wg.Wait()
}
