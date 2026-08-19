package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validBase() *AppConfig {
	return &AppConfig{
		Environment:              "development",
		ServerPort:               8888,
		DBType:                   "postgres",
		DBHost:                   "localhost",
		DBPort:                   5432,
		DBUser:                   "postgres",
		DBPassword:               "password",
		DBName:                   "boot",
		DBMaxIdleConns:           10,
		DBMaxOpenConns:           100,
		DBConnMaxLifetimeMinutes: 60,
		AccessTokenSecret:        "access-secret",
		RefreshTokenSecret:       "refresh-secret",
		JWTIssuer:                "boot",
		AccessTokenLifetime:      1,
		RefreshTokenLifetime:     168,
	}
}

func TestValidate_Baseline(t *testing.T) {
	assert.NoError(t, validBase().Validate())
}

func TestValidate_RequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		clear func(*AppConfig)
		field string
	}{
		{name: "environment", clear: func(c *AppConfig) { c.Environment = "" }, field: "APP_ENVIRONMENT"},
		{name: "server port", clear: func(c *AppConfig) { c.ServerPort = 0 }, field: "SERVER_PORT"},
		{name: "db host", clear: func(c *AppConfig) { c.DBHost = "" }, field: "DB_HOST"},
		{name: "access token secret", clear: func(c *AppConfig) { c.AccessTokenSecret = "" }, field: "ACCESS_TOKEN_SECRET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validBase()
			tt.clear(cfg)

			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.field)
		})
	}
}

func TestValidate_DisabledComponentsAreNotChecked(t *testing.T) {
	cfg := validBase()
	cfg.CacheEnabled = false
	cfg.StorageEnabled = false

	cfg.RedisAddr = ""
	cfg.MinIOEndpoint = ""
	cfg.CacheTTLJitter = 42

	assert.NoError(t, cfg.Validate())
}

func TestValidate_Cache(t *testing.T) {
	t.Run("enabled and configured", func(t *testing.T) {
		cfg := validBase()
		cfg.CacheEnabled = true
		cfg.RedisAddr = "redis:6379"
		cfg.CacheTTLJitter = 0.1

		assert.NoError(t, cfg.Validate())
	})

	t.Run("enabled without an address", func(t *testing.T) {
		cfg := validBase()
		cfg.CacheEnabled = true

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REDIS_ADDR")
	})

	t.Run("jitter out of range", func(t *testing.T) {
		for _, jitter := range []float64{-0.1, 1, 2.5} {
			cfg := validBase()
			cfg.CacheEnabled = true
			cfg.RedisAddr = "redis:6379"
			cfg.CacheTTLJitter = jitter

			err := cfg.Validate()
			require.Error(t, err, "jitter %v should be rejected", jitter)
			assert.Contains(t, err.Error(), "CACHE_TTL_JITTER")
		}
	})
}

func TestValidate_Storage(t *testing.T) {
	configured := func() *AppConfig {
		cfg := validBase()
		cfg.StorageEnabled = true
		cfg.MinIOEndpoint = "minio:9000"
		cfg.MinIOAccessKey = "minioadmin"
		cfg.MinIOSecretKey = "minioadmin"
		cfg.MinIOBucket = "boot"
		return cfg
	}

	t.Run("enabled and configured", func(t *testing.T) {
		assert.NoError(t, configured().Validate())
	})

	tests := []struct {
		name  string
		clear func(*AppConfig)
		field string
	}{
		{name: "endpoint", clear: func(c *AppConfig) { c.MinIOEndpoint = "" }, field: "MINIO_ENDPOINT"},
		{name: "access key", clear: func(c *AppConfig) { c.MinIOAccessKey = "" }, field: "MINIO_ACCESS_KEY"},
		{name: "secret key", clear: func(c *AppConfig) { c.MinIOSecretKey = "" }, field: "MINIO_SECRET_KEY"},
		{name: "bucket", clear: func(c *AppConfig) { c.MinIOBucket = "" }, field: "MINIO_BUCKET"},
	}

	for _, tt := range tests {
		t.Run("missing "+tt.name, func(t *testing.T) {
			cfg := configured()
			tt.clear(cfg)

			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.field)
		})
	}

	t.Run("public endpoint is optional", func(t *testing.T) {
		cfg := configured()
		cfg.MinIOPublicEndpoint = ""

		assert.NoError(t, cfg.Validate())
	})
}

func TestServerAddress(t *testing.T) {
	cfg := &AppConfig{ServerPort: 8888}
	assert.Equal(t, ":8888", cfg.ServerAddress())
}
