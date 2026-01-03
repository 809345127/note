package cmd

import (
	"fmt"
	"net/http"
	"os"

	"ddd/api"
	"ddd/api/health"
	apiorder "ddd/api/order"
	apiuser "ddd/api/user"
	orderapp "ddd/application/order"
	userapp "ddd/application/user"
	"ddd/config"
	orderdomain "ddd/domain/order"
	"ddd/domain/shared"
	userdomain "ddd/domain/user"
	"ddd/infrastructure/persistence/mysql"
	"ddd/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AppBuilder builds an App with customizable components
type AppBuilder struct {
	cfg          *config.Config
	controllers  []api.ControllerRegister
	middlewares  []api.MiddlewareRegister
	customRoutes []api.Route
	useDefaultDB bool
}

// NewBuilder creates a new AppBuilder
func NewBuilder(cfg *config.Config) *AppBuilder {
	return &AppBuilder{
		cfg:          cfg,
		controllers:  []api.ControllerRegister{},
		middlewares:  []api.MiddlewareRegister{},
		customRoutes: []api.Route{},
		useDefaultDB: true,
	}
}

// WithController adds a controller to the app
func (b *AppBuilder) WithController(c api.ControllerRegister) *AppBuilder {
	b.controllers = append(b.controllers, c)
	return b
}

// WithMiddleware adds a middleware to the app
func (b *AppBuilder) WithMiddleware(m api.MiddlewareRegister) *AppBuilder {
	b.middlewares = append(b.middlewares, m)
	return b
}

// WithRoute adds a custom route
func (b *AppBuilder) WithRoute(method, path string, handler gin.HandlerFunc) *AppBuilder {
	b.customRoutes = append(b.customRoutes, api.Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	})
	return b
}

// DisableDefaultDB disables the default MySQL database initialization
func (b *AppBuilder) DisableDefaultDB() *AppBuilder {
	b.useDefaultDB = false
	return b
}

// Build creates the App instance
func (b *AppBuilder) Build() *App {
	// Initialize logger
	if err := logger.Init(&b.cfg.Log, b.cfg.App.Env); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting application",
		zap.String("app", b.cfg.App.Name),
		zap.String("version", b.cfg.App.Version),
		zap.String("env", b.cfg.App.Env))

	var db *gorm.DB
	var userRepo userdomain.Repository
	var orderRepo orderdomain.Repository
	var uow shared.UnitOfWork

	// Initialize default database if enabled
	if b.useDefaultDB {
		db, userRepo, orderRepo, uow = b.initDefaultDatabase()
	}

	// Create default services
	var userService *userapp.ApplicationService
	var orderService *orderapp.ApplicationService

	if userRepo != nil {
		userService = userapp.NewApplicationService(userRepo, orderRepo, uow)
	}
	if orderRepo != nil {
		orderService = orderapp.NewApplicationService(orderRepo, userRepo, uow)
	}

	// Create default controllers if not provided
	if !b.hasHealthController() {
		b.controllers = append(b.controllers, b.getOrCreateHealthController(db))
	}
	if !b.hasUserController() && userService != nil {
		b.controllers = append(b.controllers, apiuser.NewController(userService))
	}
	if !b.hasOrderController() && orderService != nil {
		b.controllers = append(b.controllers, apiorder.NewController(orderService))
	}

	// Create router with controllers and middleware
	router := api.NewRouter(b.cfg, b.controllers, b.middlewares, b.customRoutes)
	router.SetupRoutes()

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + b.cfg.Server.Port,
		Handler:      router.GetEngine(),
		ReadTimeout:  b.cfg.Server.ReadTimeout,
		WriteTimeout: b.cfg.Server.WriteTimeout,
	}

	app := &App{
		config: b.cfg,
		router: router,
		server: server,
		db:     db,
	}

	return app
}

func (b *AppBuilder) initDefaultDatabase() (*gorm.DB, userdomain.Repository, orderdomain.Repository, shared.UnitOfWork) {
	logger.Info("Using MySQL/GORM persistence layer")

	mysqlConfig := &mysql.Config{
		Host:            b.cfg.Database.Host,
		Port:            b.cfg.Database.Port,
		Username:        b.cfg.Database.Username,
		Password:        b.cfg.Database.Password,
		Database:        b.cfg.Database.Database,
		MaxOpenConns:    b.cfg.Database.MaxOpenConns,
		MaxIdleConns:    b.cfg.Database.MaxIdleConns,
		ConnMaxLifetime: b.cfg.Database.ConnMaxLifetime,
	}

	db, err := mysqlConfig.Connect()
	if err != nil {
		logger.Fatal("Failed to connect to MySQL", zap.Error(err))
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("Failed to get underlying sql.DB", zap.Error(err))
	}
	if err := sqlDB.Ping(); err != nil {
		logger.Fatal("Failed to ping MySQL", zap.Error(err))
	}

	logger.Info("Connected to MySQL successfully")

	// Auto migration in development environment
	if b.cfg.IsDevelopment() {
		if err := mysql.AutoMigrate(db); err != nil {
			logger.Fatal("Failed to auto migrate", zap.Error(err))
		}
	}

	userRepo := mysql.NewUserRepository(db)
	orderRepo := mysql.NewOrderRepository(db)
	uow := mysql.NewUnitOfWork(db)

	return db, userRepo, orderRepo, uow
}

func (b *AppBuilder) hasUserController() bool {
	for _, c := range b.controllers {
		if _, ok := c.(*apiuser.Controller); ok {
			return true
		}
	}
	return false
}

func (b *AppBuilder) hasOrderController() bool {
	for _, c := range b.controllers {
		if _, ok := c.(*apiorder.Controller); ok {
			return true
		}
	}
	return false
}

func (b *AppBuilder) hasHealthController() bool {
	for _, c := range b.controllers {
		if _, ok := c.(*health.Controller); ok {
			return true
		}
	}
	return false
}

func (b *AppBuilder) getOrCreateHealthController(db *gorm.DB) *health.Controller {
	var healthDB interface{}
	if db != nil {
		sqlDB, _ := db.DB()
		healthDB = sqlDB
	}
	return health.NewController(b.cfg, healthDB)
}
