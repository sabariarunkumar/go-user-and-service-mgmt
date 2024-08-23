package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"userservice/internal/components/role"
	"userservice/internal/components/service"
	"userservice/internal/components/user"
	"userservice/internal/configs"
	"userservice/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	V1apiRoutePrefix = "/api/v1"
)

// APIServer contains listener info and entities which will be used by underlying route endpoints
type APIServer struct {
	addr   string
	config *configs.Config
	ctx    context.Context
	db     *gorm.DB
	wg     *sync.WaitGroup
	logger *zap.SugaredLogger
}

// NewAPIServer initializes entities for server and endpoint runtime
func NewAPIServer(ctx context.Context, wg *sync.WaitGroup,
	config *configs.Config, db *gorm.DB, logger *zap.SugaredLogger) *APIServer {

	return &APIServer{
		addr:   fmt.Sprintf(":%s", config.ServerPort),
		config: config,
		ctx:    ctx,
		wg:     wg,
		db:     db,
		logger: logger,
	}
}

// Run register routes across modules
func (s *APIServer) StartAPIServer(wg *sync.WaitGroup, serverErr chan error) *http.Server {

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	v1Apis := router.Group(V1apiRoutePrefix)

	// Use global middleware to validate JWT token
	v1Apis.Use(middleware.Authenticate(V1apiRoutePrefix, s.logger, []byte(s.config.JWTSecret)))

	userHandler := user.NewHandler(s.logger, s.config, s.db)
	userHandler.RegisterRoutes(v1Apis)

	roleHandler := role.NewHandler(s.logger, s.db)
	roleHandler.RegisterRoutes(v1Apis)

	serviceHandler := service.NewHandler(s.ctx, s.wg, s.logger, s.config, s.db)
	serviceHandler.RegisterRoutes(v1Apis)

	var server = &http.Server{Addr: ":8080", Handler: router}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			serverErr <- err
		}
	}()
	return server
}
