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
	"ddd/infrastructure/persistence/retry"
	"ddd/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AppBuilder struct {
	cfg          *config.Config
	controllers  []api.ControllerRegister
	middlewares  []api.MiddlewareRegister
	customRoutes []api.Route
}

func NewBuilder(cfg *config.Config) *AppBuilder {
	return &AppBuilder{
		cfg:          cfg,
		controllers:  []api.ControllerRegister{},
		middlewares:  []api.MiddlewareRegister{},
		customRoutes: []api.Route{},
	}
}
func (b *AppBuilder) WithController(c api.ControllerRegister) *AppBuilder {
	b.controllers = append(b.controllers, c)
	return b
}
func (b *AppBuilder) WithMiddleware(m api.MiddlewareRegister) *AppBuilder {
	b.middlewares = append(b.middlewares, m)
	return b
}
func (b *AppBuilder) WithRoute(method, path string, handler gin.HandlerFunc) *AppBuilder {
	b.customRoutes = append(b.customRoutes, api.Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	})
	return b
}

func (b *AppBuilder) Build() *App {
	if err := logger.Init(&b.cfg.Log, b.cfg.App.Env); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting application",
		zap.String("app", b.cfg.App.Name),
		zap.String("version", b.cfg.App.Version),
		zap.String("env", b.cfg.App.Env))

	db, userRepo, orderRepo, uowFactory := b.initMySQLPersistence()
	userService := userapp.NewApplicationService(userRepo, orderRepo, uowFactory)
	orderService := orderapp.NewApplicationService(orderRepo, userRepo, uowFactory)

	if !b.hasHealthController() {
		b.controllers = append(b.controllers, b.newHealthController(db))
	}
	if !b.hasUserController() {
		b.controllers = append(b.controllers, apiuser.NewController(userService))
	}
	if !b.hasOrderController() {
		b.controllers = append(b.controllers, apiorder.NewController(orderService))
	}
	router := api.NewRouter(b.cfg, b.controllers, b.middlewares, b.customRoutes)
	router.SetupRoutes()
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

func (b *AppBuilder) initMySQLPersistence() (*gorm.DB, userdomain.Repository, orderdomain.Repository, shared.UnitOfWorkFactory) {
	logger.Info("Using MySQL/GORM persistence layer")

	db, err := NewMySQLConfig(b.cfg).Connect()
	if err != nil {
		logger.Fatal("Failed to connect to MySQL", zap.Error(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("Failed to get underlying sql.DB", zap.Error(err))
	}
	if err := sqlDB.Ping(); err != nil {
		logger.Fatal("Failed to ping MySQL", zap.Error(err))
	}

	logger.Info("Connected to MySQL successfully")

	userRepo := mysql.NewUserRepository(db)
	orderRepo := mysql.NewOrderRepository(db)
	uowFactory := mysql.NewUnitOfWorkFactory(
		db,
		retry.FromAppConfig(b.cfg),
	)

	return db, userRepo, orderRepo, uowFactory
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

func (b *AppBuilder) newHealthController(db *gorm.DB) *health.Controller {
	var healthDB interface{}
	if db != nil {
		sqlDB, _ := db.DB()
		healthDB = sqlDB
	}
	return health.NewController(b.cfg, healthDB)
}
