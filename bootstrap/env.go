package bootstrap

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// AppConfig holds all configuration for the application
type AppConfig struct {
	Environment          string `mapstructure:"APP_ENVIRONMENT"`
	ServerAddress        string `mapstructure:"SERVER_ADDRESS"`
	RequestTimeout       int    `mapstructure:"REQUEST_TIMEOUT_SECONDS"`
	DatabaseHost         string `mapstructure:"DATABASE_HOST"`
	DatabasePort         string `mapstructure:"DATABASE_PORT"`
	DatabaseUser         string `mapstructure:"DATABASE_USER"`
	DatabasePassword     string `mapstructure:"DATABASE_PASSWORD"`
	DatabaseName         string `mapstructure:"DATABASE_NAME"`
	AccessTokenLifetime  int    `mapstructure:"ACCESS_TOKEN_LIFETIME_HOURS"`
	RefreshTokenLifetime int    `mapstructure:"REFRESH_TOKEN_LIFETIME_HOURS"`
	AccessTokenSecret    string `mapstructure:"ACCESS_TOKEN_SECRET"`
	RefreshTokenSecret   string `mapstructure:"REFRESH_TOKEN_SECRET"`
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
		log.Println("Application is running in development mode")
	}

	return config, nil
}
