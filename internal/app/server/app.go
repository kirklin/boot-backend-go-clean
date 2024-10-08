package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/auth"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/route"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"github.com/kirklin/boot-backend-go-clean/pkg/database/mysql"
	"github.com/kirklin/boot-backend-go-clean/pkg/database/postgres"
	"time"
)

// Application holds the core components of the application
type Application struct {
	Config *configs.AppConfig
	Router *gin.Engine
	DB     database.Database
}

// NewApplication creates and initializes a new Application instance
func NewApplication() (*Application, error) {
	config, err := configs.LoadConfig()
	if err != nil {
		return nil, err
	}

	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add timeout middleware
	//router.Use(middleware.TimeoutMiddleware(time.Duration(config.RequestTimeout) * time.Second))

	app := &Application{
		Config: config,
		Router: router,
	}

	return app, nil
}

// Initialize performs any necessary setup for the application
func (app *Application) Initialize() error {
	// Initialize database
	dbConfig := &database.Config{
		Host:     app.Config.DatabaseHost,
		Port:     app.Config.DatabasePort,
		User:     app.Config.DatabaseUser,
		Password: app.Config.DatabasePassword,
		DBName:   app.Config.DatabaseName,
		SSLMode:  app.Config.DatabaseSSLMode,
	}

	var err error
	switch app.Config.DatabaseType {
	case "postgres":
		app.DB = postgres.NewPostgresDB()
	case "mysql":
		app.DB = mysql.NewMySQLDB()
	default:
		return fmt.Errorf("unsupported database type: %s", app.Config.DatabaseType)
	}

	err = app.DB.Connect(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := database.AutoMigrate(app.DB); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	// Initialize JWT
	auth.InitJWT(app.Config.AccessTokenSecret,
		app.Config.RefreshTokenSecret,
		app.Config.JWTIssuer,
		time.Duration(app.Config.AccessTokenLifetime)*time.Hour,
		time.Duration(app.Config.RefreshTokenLifetime)*time.Hour,
	)

	// Set up routes
	route.SetupRoutes(app.Router, app.DB, app.Config)
	return nil
}

// Run starts the application
func (app *Application) Run() error {
	err := app.Router.SetTrustedProxies(nil)
	if err != nil {
		return err
	}
	return app.Router.Run(app.Config.ServerAddress)
}

// Shutdown performs any necessary cleanup before the application exits
func (app *Application) Shutdown() {
	if app.DB != nil {
		_ = app.DB.Close()
	}
}
