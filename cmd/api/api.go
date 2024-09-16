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
	addr      string
	config    *configs.Config
	ctx       context.Context
	db        *gorm.DB
	wg        *sync.WaitGroup
	logger    *zap.SugaredLogger
	errorChan chan error

	// exported for main module to use it
	Runtime *http.Server
}

// NewAPIServer initializes entities for server and endpoint runtime
func NewAPIServer(ctx context.Context, wg *sync.WaitGroup,
	config *configs.Config, db *gorm.DB, logger *zap.SugaredLogger, serverErr chan error) *APIServer {

	return &APIServer{
		addr:      fmt.Sprintf(":%s", config.ServerPort),
		config:    config,
		ctx:       ctx,
		wg:        wg,
		db:        db,
		logger:    logger,
		errorChan: serverErr,
	}
}

// Run register routes across modules
func (s *APIServer) StartAPIServer() {

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

	s.Runtime = &http.Server{Addr: ":8080", Handler: router}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.Runtime.ListenAndServe(); err != http.ErrServerClosed {
			s.errorChan <- err
		}
	}()
}
