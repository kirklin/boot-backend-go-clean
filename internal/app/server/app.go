package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/route"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"github.com/kirklin/boot-backend-go-clean/pkg/database/mysql"
	"github.com/kirklin/boot-backend-go-clean/pkg/database/postgres"
	"github.com/kirklin/boot-backend-go-clean/pkg/logger"
	snowflakeutils "github.com/kirklin/boot-backend-go-clean/pkg/utils/snowflake"
)

// Application holds the core components of the application
type Application struct {
	Config     *configs.AppConfig
	Router     *gin.Engine
	DB         database.Database
	httpServer *http.Server
}

// NewApplication creates and initializes a new Application instance
func NewApplication() (*Application, error) {
	// Lock the global timezone to UTC to enforce UTC Everywhere
	time.Local = time.UTC

	config, err := configs.LoadConfig()
	if err != nil {
		return nil, err
	}

	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Redirect Gin's logs to our custom logger
	gin.DefaultWriter = &ginLogWriter{logger: logger.GetLogger()}
	router := gin.New()

	// Register RequestID as early as possible so it's included in logs
	router.Use(middleware.RequestID())

	router.Use(gin.LoggerWithWriter(gin.DefaultWriter))
	router.Use(gin.Recovery())

	// Add global ErrorHandler middleware to format any c.Error() calls
	router.Use(middleware.ErrorHandler())

	// Add timeout middleware
	router.Use(middleware.TimeoutMiddleware(time.Duration(config.RequestTimeout) * time.Second))

	app := &Application{
		Config: config,
		Router: router,
	}

	return app, nil
}

// Initialize performs any necessary setup for the application
func (app *Application) Initialize() error {
	// Initialize Snowflake
	if err := snowflakeutils.InitNode(&snowflakeutils.Config{
		Epoch:       app.Config.SnowflakeEpoch,
		MachineBits: app.Config.SnowflakeMachineBits,
		StepBits:    app.Config.SnowflakeStepBits,
	}); err != nil {
		logger.GetLogger().Fatalf("failed to init snowflake node: %v", err)
	}

	// Initialize database
	dbConfig := &database.Config{
		Host:                   app.Config.DBHost,
		Port:                   app.Config.DBPort,
		User:                   app.Config.DBUser,
		Password:               app.Config.DBPassword,
		DBName:                 app.Config.DBName,
		SSLMode:                app.Config.DBSSLMode,
		MaxIdleConns:           app.Config.DBMaxIdleConns,
		MaxOpenConns:           app.Config.DBMaxOpenConns,
		ConnMaxLifetimeMinutes: app.Config.DBConnMaxLifetimeMinutes,
	}

	var err error
	switch app.Config.DBType {
	case "postgres":
		app.DB = postgres.NewPostgresDB()
	case "mysql":
		app.DB = mysql.NewMySQLDB()
	default:
		logger.GetLogger().Fatalf("unsupported database type: %s", app.Config.DBType)
	}

	err = app.DB.Connect(dbConfig)
	if err != nil {
		logger.GetLogger().Fatalf("failed to connect to database: %v", err)
	}

	if err := persistence.AutoMigrate(app.DB); err != nil {
		logger.GetLogger().Fatalf("failed to auto migrate: %v", err)
	}

	// Set up routes
	route.SetupRoutes(app.Router, app.DB, app.Config)
	return nil
}

// shutdownGracePeriod is the maximum time to wait for in-flight requests
// to complete during graceful shutdown.
const shutdownGracePeriod = 30 * time.Second

// Run starts the HTTP server and blocks until ctx is canceled.
// When the parent context is canceled (e.g. via signal.NotifyContext),
// it automatically performs graceful shutdown: stops accepting new connections,
// waits for in-flight requests to drain, and closes the database.
//
// This is the single lifecycle method — the caller only needs to do:
//
//	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
//	defer stop()
//	app.Run(ctx)
func (app *Application) Run(ctx context.Context) error {
	log := logger.GetLogger()

	if err := app.Router.SetTrustedProxies(nil); err != nil {
		return err
	}

	app.httpServer = &http.Server{
		Addr:    app.Config.ServerAddress(),
		Handler: app.Router,

		// Defense against slowloris and resource exhaustion attacks.
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Start the HTTP server in a goroutine so we can listen for ctx cancellation.
	serverErr := make(chan error, 1)
	go func() {
		log.Infof("HTTP server listening on %s", app.Config.ServerAddress())
		if err := app.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("HTTP server error: %w", err)
		}
		close(serverErr)
	}()

	// Block until we receive a shutdown signal or the server fails to start.
	select {
	case err := <-serverErr:
		// Server failed to start (e.g. port already in use). Clean up and return.
		app.shutdown()
		return err
	case <-ctx.Done():
		log.Info("Shutdown signal received, draining in-flight requests...")
	}

	// Graceful shutdown with a deadline.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()

	// 1. Stop accepting new connections and drain in-flight requests.
	if err := app.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Errorf("HTTP server forced to shutdown: %v", err)
	} else {
		log.Info("HTTP server drained successfully")
	}

	// 2. Close infrastructure resources (database, etc.).
	app.shutdown()

	log.Info("Application stopped")
	return nil
}

// shutdown closes all infrastructure resources.
func (app *Application) shutdown() {
	log := logger.GetLogger()
	if app.DB != nil {
		log.Info("Closing database connection...")
		if err := app.DB.Close(); err != nil {
			log.Errorf("Error closing database: %v", err)
		} else {
			log.Info("Database connection closed")
		}
	}
}

// ginLogWriter is a writer that redirects Gin's logs to our custom logger
type ginLogWriter struct {
	logger logger.Logger
}

func (w *ginLogWriter) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	ctx := context.Background()
	// Parse log level from message
	if strings.HasPrefix(message, "[GIN-debug] [WARNING]") {
		w.logger.Log(ctx, logger.WarnLevel, strings.TrimPrefix(message, "[GIN-debug] [WARNING] "), nil)
	} else if strings.HasPrefix(message, "[GIN-debug] [ERROR]") {
		w.logger.Log(ctx, logger.ErrorLevel, strings.TrimPrefix(message, "[GIN-debug] [ERROR] "), nil)
	} else if strings.HasPrefix(message, "[GIN-debug]") {
		w.logger.Log(ctx, logger.DebugLevel, strings.TrimPrefix(message, "[GIN-debug] "), nil)
	} else {
		w.logger.Log(ctx, logger.InfoLevel, message, nil)
	}

	return len(p), nil
}
