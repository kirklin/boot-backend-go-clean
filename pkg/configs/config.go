package configs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// AppConfig holds all configuration for the application
type AppConfig struct {
	Environment    string `mapstructure:"APP_ENVIRONMENT"`
	ServerPort     int    `mapstructure:"SERVER_PORT"`
	RequestTimeout int    `mapstructure:"REQUEST_TIMEOUT_SECONDS"`
	// Rate Limiting
	RateLimitPerMinute int `mapstructure:"RATE_LIMIT_PER_MINUTE"` // 每 IP 每分钟最大请求数，0 = 不限流
	// Database
	DBType                   string `mapstructure:"DB_TYPE"`
	DBHost                   string `mapstructure:"DB_HOST"`
	DBPort                   int    `mapstructure:"DB_PORT"`
	DBUser                   string `mapstructure:"DB_USER"`
	DBPassword               string `mapstructure:"DB_PASSWORD"`
	DBName                   string `mapstructure:"DB_NAME"`
	DBSSLMode                string `mapstructure:"DB_SSL_MODE"`
	DBMaxIdleConns           int    `mapstructure:"DB_MAX_IDLE_CONNS"`
	DBMaxOpenConns           int    `mapstructure:"DB_MAX_OPEN_CONNS"`
	DBConnMaxLifetimeMinutes int    `mapstructure:"DB_CONN_MAX_LIFETIME_MINUTES"`
	// JWT
	AccessTokenLifetime  int    `mapstructure:"ACCESS_TOKEN_LIFETIME_HOURS"`
	RefreshTokenLifetime int    `mapstructure:"REFRESH_TOKEN_LIFETIME_HOURS"`
	AccessTokenSecret    string `mapstructure:"ACCESS_TOKEN_SECRET"`
	RefreshTokenSecret   string `mapstructure:"REFRESH_TOKEN_SECRET"`
	JWTIssuer            string `mapstructure:"JWT_ISSUER"`
	// Snowflake
	SnowflakeEpoch       string `mapstructure:"SNOWFLAKE_EPOCH"`
	SnowflakeMachineBits int    `mapstructure:"SNOWFLAKE_MACHINE_BITS"`
	SnowflakeStepBits    int    `mapstructure:"SNOWFLAKE_STEP_BITS"`
	// AI / LLM — generic naming; the concrete framework (Eino) is an implementation detail.
	AIEnabled bool   `mapstructure:"AI_ENABLED"` // Master switch, default false
	AIModel   string `mapstructure:"AI_MODEL"`   // e.g. gpt-4o, deepseek-chat
	AIAPIKey  string `mapstructure:"AI_API_KEY"`
	AIBaseURL string `mapstructure:"AI_BASE_URL"` // Optional, for OpenAI-compatible endpoints
	// Langfuse — LLM observability
	LangfuseEnabled   bool   `mapstructure:"LANGFUSE_ENABLED"` // Master switch, default false
	LangfuseHost      string `mapstructure:"LANGFUSE_HOST"`
	LangfusePublicKey string `mapstructure:"LANGFUSE_PUBLIC_KEY"`
	LangfuseSecretKey string `mapstructure:"LANGFUSE_SECRET_KEY"`
	// Login Activity — IP geolocation and audit logging
	LoginActivityEnabled       bool `mapstructure:"LOGIN_ACTIVITY_ENABLED"`        // Enable login activity recording, default true
	LoginActivityRetentionDays int  `mapstructure:"LOGIN_ACTIVITY_RETENTION_DAYS"` // Auto-cleanup records older than N days, default 180 (6 months), 0 = keep forever
	// Cache (Redis)
	CacheEnabled               bool    `mapstructure:"CACHE_ENABLED"`
	RedisAddr                  string  `mapstructure:"REDIS_ADDR"`
	RedisPassword              string  `mapstructure:"REDIS_PASSWORD"`
	RedisDB                    int     `mapstructure:"REDIS_DB"`
	RedisPoolSize              int     `mapstructure:"REDIS_POOL_SIZE"`
	CacheKeyPrefix             string  `mapstructure:"CACHE_KEY_PREFIX"`
	CacheTTLJitter             float64 `mapstructure:"CACHE_TTL_JITTER"`
	CacheNegativeTTLSeconds    int     `mapstructure:"CACHE_NEGATIVE_TTL_SECONDS"`
	CacheRefreshTimeoutSeconds int     `mapstructure:"CACHE_REFRESH_TIMEOUT_SECONDS"`
	// Object Storage (S3-compatible)
	StorageEnabled      bool   `mapstructure:"STORAGE_ENABLED"`
	MinIOEndpoint       string `mapstructure:"MINIO_ENDPOINT"`
	MinIOPublicEndpoint string `mapstructure:"MINIO_PUBLIC_ENDPOINT"`
	MinIOAccessKey      string `mapstructure:"MINIO_ACCESS_KEY"`
	MinIOSecretKey      string `mapstructure:"MINIO_SECRET_KEY"`
	MinIOUseSSL         bool   `mapstructure:"MINIO_USE_SSL"`
	MinIOPublicUseSSL   bool   `mapstructure:"MINIO_PUBLIC_USE_SSL"`
	MinIORegion         string `mapstructure:"MINIO_REGION"`
	MinIOBucket         string `mapstructure:"MINIO_BUCKET"`
	MinIOKeyPrefix      string `mapstructure:"MINIO_KEY_PREFIX"`
	MinIOEnsureBucket   bool   `mapstructure:"MINIO_ENSURE_BUCKET"`
}

// ServerAddress returns the formatted server address
func (c *AppConfig) ServerAddress() string {
	return fmt.Sprintf(":%d", c.ServerPort)
}

// Validate checks that all required configuration fields are set.
// Returns an error listing all missing fields if any are empty.
func (c *AppConfig) Validate() error {
	var errs []error

	requireStr := func(val, name string) {
		if strings.TrimSpace(val) == "" {
			errs = append(errs, fmt.Errorf("  - %s is required but not set", name))
		}
	}
	requireInt := func(val int, name string) {
		if val == 0 {
			errs = append(errs, fmt.Errorf("  - %s is required but not set (or is 0)", name))
		}
	}

	// ---- 应用基础 ----
	requireStr(c.Environment, "APP_ENVIRONMENT")
	requireInt(c.ServerPort, "SERVER_PORT")

	// ---- 数据库 ----
	requireStr(c.DBType, "DB_TYPE")
	requireStr(c.DBHost, "DB_HOST")
	requireInt(c.DBPort, "DB_PORT")
	requireStr(c.DBUser, "DB_USER")
	requireStr(c.DBPassword, "DB_PASSWORD")
	requireStr(c.DBName, "DB_NAME")
	requireInt(c.DBMaxIdleConns, "DB_MAX_IDLE_CONNS")
	requireInt(c.DBMaxOpenConns, "DB_MAX_OPEN_CONNS")
	requireInt(c.DBConnMaxLifetimeMinutes, "DB_CONN_MAX_LIFETIME_MINUTES")

	// ---- JWT ----
	requireStr(c.AccessTokenSecret, "ACCESS_TOKEN_SECRET")
	requireStr(c.RefreshTokenSecret, "REFRESH_TOKEN_SECRET")
	requireStr(c.JWTIssuer, "JWT_ISSUER")
	requireInt(c.AccessTokenLifetime, "ACCESS_TOKEN_LIFETIME_HOURS")
	requireInt(c.RefreshTokenLifetime, "REFRESH_TOKEN_LIFETIME_HOURS")

	if c.CacheEnabled {
		requireStr(c.RedisAddr, "REDIS_ADDR")
		if c.CacheTTLJitter < 0 || c.CacheTTLJitter >= 1 {
			errs = append(errs, fmt.Errorf("  - CACHE_TTL_JITTER must be in [0, 1), got %v", c.CacheTTLJitter))
		}
		if c.CacheNegativeTTLSeconds < 0 {
			errs = append(errs, fmt.Errorf("  - CACHE_NEGATIVE_TTL_SECONDS must not be negative, got %d", c.CacheNegativeTTLSeconds))
		}
	}

	if c.StorageEnabled {
		requireStr(c.MinIOEndpoint, "MINIO_ENDPOINT")
		requireStr(c.MinIOAccessKey, "MINIO_ACCESS_KEY")
		requireStr(c.MinIOSecretKey, "MINIO_SECRET_KEY")
		requireStr(c.MinIOBucket, "MINIO_BUCKET")
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n%w", errors.Join(errs...))
	}
	return nil
}

// LoadConfig reads the configuration from .env file and environment variables
func LoadConfig() (*AppConfig, error) {
	config := &AppConfig{}

	viper.SetConfigFile(".env")
	viper.AutomaticEnv() // Allow true environment variables (e.g., from Docker) to override .env configs

	// Sensible defaults — can be overridden by .env or env vars
	viper.SetDefault("LOGIN_ACTIVITY_ENABLED", true)
	viper.SetDefault("LOGIN_ACTIVITY_RETENTION_DAYS", 180) // 6 months
	viper.SetDefault("CACHE_TTL_JITTER", 0.1)
	viper.SetDefault("CACHE_NEGATIVE_TTL_SECONDS", 30)
	viper.SetDefault("CACHE_REFRESH_TIMEOUT_SECONDS", 10)

	// Ignore error if .env doesn't exist, as we might rely entirely on env vars
	_ = viper.ReadInConfig()

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}
