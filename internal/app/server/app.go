package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/ai"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/auth"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/geoip"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/metrics"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/route"
	"github.com/kirklin/boot-backend-go-clean/internal/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/cache"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"github.com/kirklin/boot-backend-go-clean/pkg/database/mysql"
	"github.com/kirklin/boot-backend-go-clean/pkg/database/postgres"
	"github.com/kirklin/boot-backend-go-clean/pkg/logger"
	"github.com/kirklin/boot-backend-go-clean/pkg/storage"
	snowflakeutils "github.com/kirklin/boot-backend-go-clean/pkg/utils/snowflake"
)

// Application holds the core components of the application
type Application struct {
	Config        *configs.AppConfig
	Router        *gin.Engine
	DB            database.Database
	Cache         cache.Cache
	Storage       storage.Storage
	httpServer    *http.Server
	geoIPResolver gateway.GeoIPResolver // optional – initialised when GeoIP data files are available
	aiProvider    *ai.Provider          // optional – initialised when AI_ENABLED=true
	langfuseFlush func()                // optional – flushes pending Langfuse traces on shutdown
	stopCleanup   chan struct{}         // signals the login activity cleanup goroutine to stop
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

	// Redirect Gin's internal debug/warning logs to our logger
	gin.DefaultWriter = &ginLogWriter{logger: logger.GetLogger()}
	router := gin.New()

	// Register RequestID as early as possible so it's included in access logs
	router.Use(middleware.RequestIDMiddleware())

	// Structured access log replaces gin.Logger() — outputs JSON fields
	// (method, path, status, latency_ms, client_ip, request_id, user_id, etc.)
	router.Use(middleware.AccessLogMiddleware(logger.GetLogger()))
	router.Use(gin.Recovery())

	// Add global ErrorHandler middleware to format any c.Error() calls
	router.Use(middleware.ErrorHandlerMiddleware())

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

	// Composition Root: build all dependencies in layer order.
	// Only app.Initialize knows about concrete implementations;
	// the route layer receives interfaces and pre-built controllers.

	// Layer 1 — Infrastructure (depends on config/db)
	if err := app.initCache(); err != nil {
		return err
	}
	if err := app.initStorage(context.Background()); err != nil {
		return err
	}

	tokenBlacklist := auth.NewTokenBlacklist()
	authenticator := auth.NewJWTAuthenticator(
		app.Config.AccessTokenSecret,
		app.Config.RefreshTokenSecret,
		app.Config.JWTIssuer,
		time.Duration(app.Config.AccessTokenLifetime)*time.Hour,
		time.Duration(app.Config.RefreshTokenLifetime)*time.Hour,
		tokenBlacklist,
	)

	// GeoIP Resolver (optional — graceful degradation if data files are missing)
	resolver, err := geoip.NewResolver("data/ip2region_v4.xdb", "data/ip2region_v6.xdb")
	if err != nil {
		logger.GetLogger().Warnf("GeoIP: %v (IP geolocation disabled)", err)
	} else {
		app.geoIPResolver = resolver
		logger.GetLogger().Info("GeoIP resolver initialized")
	}

	// Layer 2 — Repositories (depend on db)
	userRepo := persistence.NewUserRepository(app.DB)
	loginActivityRepo := persistence.NewLoginActivityRepository(app.DB)
	txManager := persistence.NewTxManager(app.DB)

	// Layer 3 — Use Cases (depend on interfaces, not concrete types)
	authUseCase := usecase.NewAuthUseCase(userRepo, authenticator, txManager, loginActivityRepo, app.geoIPResolver, app.Config)
	userUseCase := usecase.NewUserUseCase(userRepo, app.Cache)

	// Layer 4 — Controllers (depend on use case interfaces)
	authCtrl := controller.NewAuthController(authUseCase)
	userCtrl := controller.NewUserController(userUseCase)
	infraCtrl := controller.NewInfraController(app.DB, app.Cache, app.Storage, app.Config)

	// Set up routes — Router holds shared deps, each register method receives its own controller
	router := route.NewRouter(authenticator, app.Config)
	router.Setup(app.Router, authCtrl, userCtrl, infraCtrl)

	// ── Langfuse Observability (optional, independent of AI) ────────────
	if app.Config.LangfuseEnabled {
		flush, err := ai.InitLangfuse(&ai.LangfuseConfig{
			Host:      app.Config.LangfuseHost,
			PublicKey: app.Config.LangfusePublicKey,
			SecretKey: app.Config.LangfuseSecretKey,
		})
		if err != nil {
			return fmt.Errorf("failed to init langfuse: %w", err)
		}
		app.langfuseFlush = flush

		logger.GetLogger().Info("Langfuse tracing enabled")
	}

	// ── AI Infrastructure (optional) ────────────────────────────────────
	if app.Config.AIEnabled {
		provider, err := ai.NewProvider(context.Background(), &ai.ProviderConfig{
			Model:   app.Config.AIModel,
			APIKey:  app.Config.AIAPIKey,
			BaseURL: app.Config.AIBaseURL,
		})
		if err != nil {
			return fmt.Errorf("failed to init AI provider: %w", err)
		}
		app.aiProvider = provider
		logger.GetLogger().Infof("AI infrastructure initialised (model=%s, langfuse=%v)",
			app.Config.AIModel, app.Config.LangfuseEnabled)
	}

	if err := prometheus.Register(metrics.NewCacheCollector(app.Cache)); err != nil {
		var already prometheus.AlreadyRegisteredError
		if !errors.As(err, &already) {
			return fmt.Errorf("failed to register cache metrics: %w", err)
		}
	}

	// ── Login Activity Cleanup (optional) ───────────────────────────────
	if app.Config.LoginActivityEnabled && app.Config.LoginActivityRetentionDays > 0 {
		app.startLoginActivityCleanup(loginActivityRepo)
	}

	return nil
}

func (app *Application) initCache() error {
	if !app.Config.CacheEnabled {
		app.Cache = cache.NewNoop()
		logger.GetLogger().Info("Cache disabled (using no-op cache)")
		return nil
	}

	client, err := cache.NewRedis(cache.Config{
		Addr:           app.Config.RedisAddr,
		Password:       app.Config.RedisPassword,
		DB:             app.Config.RedisDB,
		PoolSize:       app.Config.RedisPoolSize,
		KeyPrefix:      app.Config.CacheKeyPrefix,
		TTLJitter:      app.Config.CacheTTLJitter,
		NegativeTTL:    time.Duration(app.Config.CacheNegativeTTLSeconds) * time.Second,
		RefreshTimeout: time.Duration(app.Config.CacheRefreshTimeoutSeconds) * time.Second,

		OnError: func(key string, err error) {
			logger.GetLogger().Warnf("Cache: %v (key=%s)", err, key)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to init cache: %w", err)
	}

	app.Cache = client
	logger.GetLogger().Infof("Cache initialized (redis=%s, db=%d, jitter=%v, negative_ttl=%ds)",
		app.Config.RedisAddr, app.Config.RedisDB, app.Config.CacheTTLJitter, app.Config.CacheNegativeTTLSeconds)
	return nil
}

func (app *Application) initStorage(ctx context.Context) error {
	if !app.Config.StorageEnabled {
		logger.GetLogger().Info("Object storage disabled")
		return nil
	}

	store, err := storage.NewMinIO(ctx, storage.Config{
		Endpoint:       app.Config.MinIOEndpoint,
		PublicEndpoint: app.Config.MinIOPublicEndpoint,
		UseSSL:         app.Config.MinIOUseSSL,
		PublicUseSSL:   app.Config.MinIOPublicUseSSL,
		AccessKey:      app.Config.MinIOAccessKey,
		SecretKey:      app.Config.MinIOSecretKey,
		Region:         app.Config.MinIORegion,
		Bucket:         app.Config.MinIOBucket,
		KeyPrefix:      app.Config.MinIOKeyPrefix,
		EnsureBucket:   app.Config.MinIOEnsureBucket,
	})
	if err != nil {
		return fmt.Errorf("failed to init object storage: %w", err)
	}

	app.Storage = store
	logger.GetLogger().Infof("Object storage initialized (endpoint=%s, bucket=%s)",
		app.Config.MinIOEndpoint, app.Config.MinIOBucket)
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

	// Configure proxy trust for correct client IP detection.
	// Request chain: Client → (Cloudflare) → Nginx → Docker → Go App
	//
	// RemoteIPHeaders defines the priority order for reading the real client IP.
	// If behind Cloudflare, CF-Connecting-IP is the most reliable source.
	// If not behind Cloudflare, X-Real-IP / X-Forwarded-For from nginx will be used.
	app.Router.RemoteIPHeaders = []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"}

	// Trust private network CIDRs (Docker bridge, Docker Desktop, localhost).
	// This allows Gin's c.ClientIP() to read the real IP from the headers above
	// instead of always returning the nginx/Docker gateway IP.
	if err := app.Router.SetTrustedProxies([]string{
		"10.0.0.0/8",     // Private Class A
		"172.16.0.0/12",  // Private Class B (includes Docker bridge 172.x.x.x)
		"192.168.0.0/16", // Private Class C (includes Docker Desktop 192.168.65.x)
		"127.0.0.0/8",    // Localhost
		"fc00::/7",       // IPv6 unique local addresses
	}); err != nil {
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

// shutdown closes all infrastructure resources in reverse initialization order.
func (app *Application) shutdown() {
	log := logger.GetLogger()

	// Flush Langfuse traces before closing AI provider so that
	// any in-flight trace data is uploaded.
	if app.langfuseFlush != nil {
		log.Info("Flushing Langfuse traces...")
		app.langfuseFlush()
		log.Info("Langfuse traces flushed")
	}

	if app.aiProvider != nil {
		log.Info("Closing AI provider...")
		if err := app.aiProvider.Close(); err != nil {
			log.Errorf("Error closing AI provider: %v", err)
		} else {
			log.Info("AI provider closed")
		}
	}

	if app.Cache != nil {
		log.Info("Closing cache...")
		if err := app.Cache.Close(); err != nil {
			log.Errorf("Error closing cache: %v", err)
		} else {
			log.Info("Cache closed")
		}
	}

	if app.stopCleanup != nil {
		log.Info("Stopping login activity cleanup...")
		close(app.stopCleanup)
		log.Info("Login activity cleanup stopped")
	}

	if app.geoIPResolver != nil {
		log.Info("Closing GeoIP resolver...")
		if err := app.geoIPResolver.Close(); err != nil {
			log.Errorf("Error closing GeoIP resolver: %v", err)
		} else {
			log.Info("GeoIP resolver closed")
		}
	}

	if app.DB != nil {
		log.Info("Closing database connection...")
		if err := app.DB.Close(); err != nil {
			log.Errorf("Error closing database: %v", err)
		} else {
			log.Info("Database connection closed")
		}
	}
}

// startLoginActivityCleanup runs a background goroutine that periodically
// deletes login activity records older than the configured retention period.
// Runs once immediately on startup, then every 24 hours.
func (app *Application) startLoginActivityCleanup(repo repository.LoginActivityRepository) {
	app.stopCleanup = make(chan struct{})
	log := logger.GetLogger()
	retention := time.Duration(app.Config.LoginActivityRetentionDays) * 24 * time.Hour

	log.Infof("Login activity cleanup enabled (retention=%d days)", app.Config.LoginActivityRetentionDays)

	go func() {
		// Run cleanup once on startup
		app.cleanupLoginActivities(repo, retention)

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				app.cleanupLoginActivities(repo, retention)
			case <-app.stopCleanup:
				return
			}
		}
	}()
}

func (app *Application) cleanupLoginActivities(repo repository.LoginActivityRepository, retention time.Duration) {
	log := logger.GetLogger()
	before := time.Now().Add(-retention)
	deleted, err := repo.DeleteBefore(context.Background(), before)
	if err != nil {
		log.Errorf("Login activity cleanup failed: %v", err)
		return
	}
	if deleted > 0 {
		log.Infof("Login activity cleanup: deleted %d records older than %s", deleted, before.Format(time.DateOnly))
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
