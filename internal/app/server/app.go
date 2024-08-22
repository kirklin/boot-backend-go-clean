package server

import (
	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/route"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
)

// Application holds the core components of the application
type Application struct {
	Config *configs.AppConfig
	Router *gin.Engine
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

	app := &Application{
		Config: config,
		Router: router,
	}

	return app, nil
}

// Initialize performs any necessary setup for the application
func (app *Application) Initialize() error {
	// Set up routes
	route.SetupRoutes(app.Router)
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
	// You can add shutdown logic here if needed
}
