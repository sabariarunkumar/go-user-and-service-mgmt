package configs

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config represents runtime config accessible by application modules.
type Config struct {
	ServerPort              string
	DBHost                  string
	DBPort                  int64
	DBUser                  string
	DBName                  string
	DBPassword              string
	DBConnTimeout           int64
	DBSlowQueryLogThreshold int64
	DBMaxConnIdleTime       int64
	DBMaxOpenConn           int64
	LogLevel                string
	JWTSecret               string
	JWTExpirationInSeconds  int64
}

// InitConfig initializes runtime config.
func InitConfig(envFile string) (*Config, error) {
	// if envfile is mentioned, and doesn't point to a valid location.
	if envFile != "" {
		err := godotenv.Load(envFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load env file from configured path %v", err)
		}
	}

	return &Config{
		ServerPort: getEnv("PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnvAsInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBName:     getEnv("DB_NAME", "userservice"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),

		// Effectively setting and consuming 20 connection active 30mins in high traffic environment.
		DBConnTimeout:           getEnvAsInt("DB_CONN_TIMEOUT_SEC", 10),
		DBMaxConnIdleTime:       getEnvAsInt("DB_MAX_CONN_IDLE_TIME_SEC", 1800),
		DBMaxOpenConn:           getEnvAsInt("DB_MAX_OPEN_CONN", 20),
		DBSlowQueryLogThreshold: getEnvAsInt("DB_SLOW_QUERY_LOG_THRESHOLD_SEC", 2),

		LogLevel: getEnv("LOG_LEVEL", "info"),
		// Secret is preferred to be sent as environment variable.
		JWTSecret:              getEnv("JWT_SECRET", "userservice123"),
		JWTExpirationInSeconds: getEnvAsInt("JWT_EXPIRATION_IN_SECONDS", 900),
	}, nil
}

// getEnv gets the env by key or return the default.
func getEnv(key, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultVal
}

// getEnvAsInt gets the env by key or use the default and convert into int64.
func getEnvAsInt(key string, defaultVal int64) int64 {
	if strValue, ok := os.LookupEnv(key); ok {
		intValue, err := strconv.ParseInt(strValue, 10, 64)
		if err != nil {
			return defaultVal
		}
		return intValue
	}
	return defaultVal
}
