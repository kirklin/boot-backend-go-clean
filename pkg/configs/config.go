package configs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kirklin/boot-backend-go-clean/pkg/logger"
	"github.com/spf13/viper"
)

// AppConfig holds all configuration for the application
type AppConfig struct {
	Environment    string `mapstructure:"APP_ENVIRONMENT"`
	AppPort        int    `mapstructure:"APP_PORT"`
	RequestTimeout int    `mapstructure:"REQUEST_TIMEOUT_SECONDS"`
	// Rate Limiting
	RateLimitPerMinute int `mapstructure:"RATE_LIMIT_PER_MINUTE"` // 每 IP 每分钟最大请求数，0 = 不限流
	// Database
	DatabaseType     string `mapstructure:"DATABASE_TYPE"`
	DatabaseHost     string `mapstructure:"DATABASE_HOST"`
	DatabasePort     int    `mapstructure:"DATABASE_PORT"`
	DatabaseUser     string `mapstructure:"DATABASE_USER"`
	DatabasePassword string `mapstructure:"DATABASE_PASSWORD"`
	DatabaseName     string `mapstructure:"DATABASE_NAME"`
	DatabaseSSLMode  string `mapstructure:"DATABASE_SSL_MODE"`
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
}

// ListenAddr returns the address string for gin to listen on, e.g. ":8888"
func (c *AppConfig) ListenAddr() string {
	return fmt.Sprintf(":%d", c.AppPort)
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
	requireInt(c.AppPort, "APP_PORT")

	// ---- 数据库 ----
	requireStr(c.DatabaseType, "DATABASE_TYPE")
	requireStr(c.DatabaseHost, "DATABASE_HOST")
	requireInt(c.DatabasePort, "DATABASE_PORT")
	requireStr(c.DatabaseUser, "DATABASE_USER")
	requireStr(c.DatabasePassword, "DATABASE_PASSWORD")
	requireStr(c.DatabaseName, "DATABASE_NAME")

	// ---- JWT ----
	requireStr(c.AccessTokenSecret, "ACCESS_TOKEN_SECRET")
	requireStr(c.RefreshTokenSecret, "REFRESH_TOKEN_SECRET")
	requireStr(c.JWTIssuer, "JWT_ISSUER")
	requireInt(c.AccessTokenLifetime, "ACCESS_TOKEN_LIFETIME_HOURS")
	requireInt(c.RefreshTokenLifetime, "REFRESH_TOKEN_LIFETIME_HOURS")

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

	if err := viper.ReadInConfig(); err != nil {
		// Ignore error if .env doesn't exist, as we might rely entirely on env vars
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.Environment == "development" {
		logger.GetLogger().Info("Application is running in development mode")
	}

	return config, nil
}
