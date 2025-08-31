package configs

import (
	"fmt"

	"github.com/kirklin/boot-backend-go-clean/pkg/logger"
	"github.com/spf13/viper"
)

// AppConfig holds all configuration for the application
type AppConfig struct {
	Environment    string `mapstructure:"APP_ENVIRONMENT"`
	ServerAddress  string `mapstructure:"SERVER_ADDRESS"`
	RequestTimeout int    `mapstructure:"REQUEST_TIMEOUT_SECONDS"`
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
}

// LoadConfig reads the configuration from .env file and environment variables
func LoadConfig() (*AppConfig, error) {
	config := &AppConfig{}

	viper.SetConfigFile(".env")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read .env file: %w", err)
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	if config.Environment == "development" {
		logger.GetLogger().Info("Application is running in development mode")
	}

	return config, nil
}
