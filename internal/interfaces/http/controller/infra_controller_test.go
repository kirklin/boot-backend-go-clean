package controller

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/kirklin/boot-backend-go-clean/pkg/cache"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"github.com/kirklin/boot-backend-go-clean/pkg/storage"
)

type stubDatabase struct {
	db *gorm.DB
}

func (s stubDatabase) Connect(*database.Config) error { return nil }
func (s stubDatabase) Close() error                   { return nil }
func (s stubDatabase) DB() *gorm.DB                   { return s.db }

type stubCache struct {
	cache.Cache
	pingErr error
}

func (s stubCache) Ping(context.Context) error { return s.pingErr }

type stubStorage struct {
	storage.Storage
	pingErr error
}

func (s stubStorage) Ping(context.Context) error { return s.pingErr }

type pingConnector struct{ err error }

func (c pingConnector) Connect(context.Context) (driver.Conn, error) { return pingConn(c), nil }
func (c pingConnector) Driver() driver.Driver                        { return nil }

type pingConn struct{ err error }

func (c pingConn) Ping(context.Context) error { return c.err }
func (c pingConn) Close() error               { return nil }
func (c pingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}
func (c pingConn) Begin() (driver.Tx, error) { return nil, errors.New("not supported") }

func healthyDatabase(t *testing.T) database.Database {
	t.Helper()

	sqlDB := sql.OpenDB(pingConnector{})
	t.Cleanup(func() { _ = sqlDB.Close() })

	return stubDatabase{db: &gorm.DB{Config: &gorm.Config{ConnPool: sqlDB}}}
}

func brokenDatabase(t *testing.T) database.Database {
	t.Helper()

	sqlDB := sql.OpenDB(pingConnector{err: errors.New("connection refused")})
	t.Cleanup(func() { _ = sqlDB.Close() })

	return stubDatabase{db: &gorm.DB{Config: &gorm.Config{ConnPool: sqlDB}}}
}

type readyResult struct {
	status int
	checks map[string]struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
}

func callReady(t *testing.T, h *InfraController) readyResult {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

	h.Ready(c)

	var body struct {
		Data map[string]struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	return readyResult{status: recorder.Code, checks: body.Data}
}

func TestReady_AllDependenciesUp(t *testing.T) {
	config := &configs.AppConfig{Environment: "development", CacheEnabled: true}
	h := NewInfraController(healthyDatabase(t), stubCache{}, stubStorage{}, config)

	got := callReady(t, h)

	assert.Equal(t, http.StatusOK, got.status)
	assert.Equal(t, "up", got.checks["database"].Status)
	assert.Equal(t, "up", got.checks["cache"].Status)
	assert.Equal(t, "up", got.checks["storage"].Status)
}

func TestReady_DatabaseDown(t *testing.T) {
	config := &configs.AppConfig{Environment: "development"}
	h := NewInfraController(brokenDatabase(t), cache.NewNoop(), nil, config)

	got := callReady(t, h)

	assert.Equal(t, http.StatusServiceUnavailable, got.status)
	assert.Equal(t, "down", got.checks["database"].Status)
	assert.Contains(t, got.checks["database"].Error, "connection refused")
}

func TestReady_CacheDown(t *testing.T) {
	config := &configs.AppConfig{Environment: "development", CacheEnabled: true}
	h := NewInfraController(
		healthyDatabase(t),
		stubCache{pingErr: errors.New("redis unreachable")},
		nil,
		config,
	)

	got := callReady(t, h)

	assert.Equal(t, http.StatusServiceUnavailable, got.status)
	assert.Equal(t, "up", got.checks["database"].Status)
	assert.Equal(t, "down", got.checks["cache"].Status)
	assert.Contains(t, got.checks["cache"].Error, "redis unreachable")
}

func TestReady_StorageDown(t *testing.T) {
	config := &configs.AppConfig{Environment: "development"}
	h := NewInfraController(
		healthyDatabase(t),
		cache.NewNoop(),
		stubStorage{pingErr: errors.New("bucket missing")},
		config,
	)

	got := callReady(t, h)

	assert.Equal(t, http.StatusServiceUnavailable, got.status)
	assert.Equal(t, "down", got.checks["storage"].Status)
}

func TestReady_DisabledCacheIsNotReported(t *testing.T) {
	config := &configs.AppConfig{Environment: "development", CacheEnabled: false}
	h := NewInfraController(healthyDatabase(t), cache.NewNoop(), nil, config)

	got := callReady(t, h)

	assert.Equal(t, http.StatusOK, got.status)
	assert.NotContains(t, got.checks, "cache")
}

func TestReady_DisabledStorageIsNotReported(t *testing.T) {
	config := &configs.AppConfig{Environment: "development"}
	h := NewInfraController(healthyDatabase(t), cache.NewNoop(), nil, config)

	got := callReady(t, h)

	assert.Equal(t, http.StatusOK, got.status)
	assert.NotContains(t, got.checks, "storage")
}

func TestReady_ProductionMasksErrors(t *testing.T) {
	config := &configs.AppConfig{Environment: "production", CacheEnabled: true}
	h := NewInfraController(
		healthyDatabase(t),
		stubCache{pingErr: errors.New("dial tcp 10.0.1.7:6379: connect: connection refused")},
		nil,
		config,
	)

	got := callReady(t, h)

	assert.Equal(t, http.StatusServiceUnavailable, got.status)
	assert.Equal(t, "service unavailable", got.checks["cache"].Error)
	assert.NotContains(t, got.checks["cache"].Error, "10.0.1.7")
}

func TestReady_ResultsAreCached(t *testing.T) {
	config := &configs.AppConfig{Environment: "development", CacheEnabled: true}
	counting := &countingCache{}
	h := NewInfraController(healthyDatabase(t), counting, nil, config)

	for range 5 {
		require.Equal(t, http.StatusOK, callReady(t, h).status)
	}

	assert.Equal(t, 1, counting.pings, "five requests within the TTL should cost one probe")
}

type countingCache struct {
	cache.Cache
	pings int
}

func (c *countingCache) Ping(context.Context) error {
	c.pings++
	return nil
}

func TestLive_DoesNotTouchDependencies(t *testing.T) {
	config := &configs.AppConfig{Environment: "development", CacheEnabled: true}
	counting := &countingCache{}

	h := NewInfraController(brokenDatabase(t), counting, nil, config)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/live", nil)

	h.Live(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Zero(t, counting.pings)
}
