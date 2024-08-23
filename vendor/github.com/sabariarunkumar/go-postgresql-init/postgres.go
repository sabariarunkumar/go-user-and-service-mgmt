package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// getDBLogger sets GORM DB logger properties.
func getDBLogger(slowThresholdInSec int64, logLevel logger.LogLevel) logger.Interface {
	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Duration(slowThresholdInSec) * time.Second,
			LogLevel:      logLevel,
			Colorful:      false,
		},
	)
}

// InitDB initializes postgres driver with configured performance parameters.
func InitDB(
	host string,
	port int64,
	user string,
	name string,
	pass string,
	appName string,
	connTimeout int64,
	connMaxIdleTime int64,
	connMaxOpenConn int64,
	logSlowQueryThresholdInSec int64,
	logLevel logger.LogLevel,
) (*gorm.DB, error) {

	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s "+
		"application_name=%s sslmode=disable connect_timeout=%d",
		host, port, user, name, pass, appName, connTimeout)

	db, err := gorm.Open(
		postgres.New(
			postgres.Config{
				DSN:                  connStr,
				PreferSimpleProtocol: true,
			}),
		&gorm.Config{
			Logger: getDBLogger(logSlowQueryThresholdInSec, logLevel),
		},
	)

	if err != nil {
		return nil, err
	}
	commonDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	commonDB.SetConnMaxIdleTime(time.Duration(connMaxIdleTime * int64(time.Second)))
	commonDB.SetMaxOpenConns(int(connMaxOpenConn))
	return db, nil
}
