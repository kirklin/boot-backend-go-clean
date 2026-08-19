package storage

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinIOStorage_Stat(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})

	modified := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	fake.seed("bucket/avatars/7.png", fakeObject{
		body:         "not-really-a-png",
		contentType:  "image/png",
		etag:         "abc123",
		lastModified: modified,
	})

	info, err := s.Stat(ctx, "avatars/7.png")
	require.NoError(t, err)

	assert.Equal(t, "avatars/7.png", info.Key, "Stat reports the bare key the caller passed")
	assert.EqualValues(t, len("not-really-a-png"), info.Size)
	assert.Equal(t, "image/png", info.ContentType)
	assert.Equal(t, "abc123", info.ETag)
	assert.True(t, info.LastModified.Equal(modified))
}

func TestMinIOStorage_StatMissing(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage(t, newFakeS3(t), Config{})

	_, err := s.Stat(ctx, "absent")
	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestMinIOStorage_Get(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	fake.seed("bucket/notes.txt", fakeObject{body: "hello", contentType: "text/plain"})

	reader, err := s.Get(ctx, "notes.txt")
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(body))
}

func TestMinIOStorage_GetMissingFailsBeforeReading(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage(t, newFakeS3(t), Config{})

	reader, err := s.Get(ctx, "absent")
	assert.ErrorIs(t, err, ErrObjectNotFound)
	assert.Nil(t, reader, "no reader should be handed back on failure")
}

func TestMinIOStorage_Put(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})

	body := "file contents"
	err := s.Put(ctx, "uploads/a.txt", strings.NewReader(body), int64(len(body)),
		WithContentType("text/plain"),
		WithMetadata(map[string]string{"owner": "kirk"}),
	)
	require.NoError(t, err)

	puts := fake.recordedPuts()
	require.Len(t, puts, 1)
	assert.Equal(t, "bucket/uploads/a.txt", puts[0].path)
	assert.Equal(t, "text/plain", puts[0].contentType)
	assert.Equal(t, "kirk", puts[0].metadata["owner"])
}

func TestMinIOStorage_PutWithoutOptions(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})

	require.NoError(t, s.Put(ctx, "a.bin", strings.NewReader("x"), 1))

	puts := fake.recordedPuts()
	require.Len(t, puts, 1)
	assert.Equal(t, "bucket/a.bin", puts[0].path)
}

func TestMinIOStorage_DeleteIsIdempotent(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	fake.seed("bucket/gone.txt", fakeObject{body: "x"})

	require.NoError(t, s.Delete(ctx, "gone.txt"))
	require.NoError(t, s.Delete(ctx, "gone.txt"), "deleting an absent key must succeed")

	assert.Equal(t, []string{"bucket/gone.txt", "bucket/gone.txt"}, fake.deletedPaths())
}

func TestMinIOStorage_KeyPrefix(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{KeyPrefix: "staging"})
	fake.seed("bucket/staging/notes.txt", fakeObject{body: "hello"})

	info, err := s.Stat(ctx, "notes.txt")
	require.NoError(t, err)
	assert.Equal(t, "notes.txt", info.Key, "the prefix is an implementation detail")

	require.NoError(t, s.Put(ctx, "new.txt", strings.NewReader("x"), 1))
	assert.Equal(t, "bucket/staging/new.txt", fake.recordedPuts()[0].path)
}

func TestMinIOStorage_Ping(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage(t, newFakeS3(t), Config{})

	assert.NoError(t, s.Ping(ctx))
}

func TestMinIOStorage_PresignUsesPublicEndpoint(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{PublicEndpoint: "cdn.example.com"})
	fake.seed("bucket/avatars/7.png", fakeObject{body: "x"})

	t.Run("get", func(t *testing.T) {
		raw, err := s.PresignGet(ctx, "avatars/7.png", time.Hour)
		require.NoError(t, err)

		parsed, err := url.Parse(raw)
		require.NoError(t, err)
		assert.Equal(t, "cdn.example.com", parsed.Host)
		assert.Equal(t, "/bucket/avatars/7.png", parsed.Path)
		assert.NotEmpty(t, parsed.Query().Get("X-Amz-Signature"))
	})

	t.Run("put", func(t *testing.T) {
		raw, err := s.PresignPut(ctx, "uploads/new.bin", time.Hour)
		require.NoError(t, err)

		parsed, err := url.Parse(raw)
		require.NoError(t, err)
		assert.Equal(t, "cdn.example.com", parsed.Host)
		assert.Equal(t, "/bucket/uploads/new.bin", parsed.Path)
		assert.NotEmpty(t, parsed.Query().Get("X-Amz-Signature"))
	})
}

func TestMinIOStorage_PresignFallsBackToEndpoint(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})

	raw, err := s.PresignPut(ctx, "a.bin", time.Hour)
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, strings.TrimPrefix(fake.server.URL, "http://"), parsed.Host)
}

func TestMinIOStorage_PresignGetMissing(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage(t, newFakeS3(t), Config{})

	_, err := s.PresignGet(ctx, "absent", time.Hour)
	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestMinIOStorage_PresignPutDoesNotRequireAnObject(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage(t, newFakeS3(t), Config{})

	raw, err := s.PresignPut(ctx, "brand/new.bin", time.Hour)
	require.NoError(t, err)
	assert.Contains(t, raw, "brand/new.bin")
}

func TestMinIOStorage_PresignExpiryBounds(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	fake.seed("bucket/k", fakeObject{body: "x"})

	tests := []struct {
		name    string
		expiry  time.Duration
		wantErr bool
	}{
		{name: "one hour", expiry: time.Hour},
		{name: "at the seven day limit", expiry: presignExpiryLimit},
		{name: "zero", expiry: 0, wantErr: true},
		{name: "negative", expiry: -time.Second, wantErr: true},
		{name: "beyond the limit", expiry: presignExpiryLimit + time.Second, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.PresignGet(ctx, "k", tt.expiry)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	complete := Config{
		Endpoint:  "minio:9000",
		AccessKey: "key",
		SecretKey: "secret",
		Bucket:    "bucket",
	}

	t.Run("complete", func(t *testing.T) {
		assert.NoError(t, complete.validate())
	})

	tests := []struct {
		name  string
		clear func(*Config)
		field string
	}{
		{name: "missing endpoint", clear: func(c *Config) { c.Endpoint = "" }, field: "Endpoint"},
		{name: "missing access key", clear: func(c *Config) { c.AccessKey = "" }, field: "AccessKey"},
		{name: "missing secret key", clear: func(c *Config) { c.SecretKey = "" }, field: "SecretKey"},
		{name: "missing bucket", clear: func(c *Config) { c.Bucket = "" }, field: "Bucket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := complete
			tt.clear(&cfg)

			err := cfg.validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.field, "the message should name the missing field")
		})
	}

	t.Run("names every missing field at once", func(t *testing.T) {
		err := Config{}.validate()
		require.Error(t, err)
		for _, field := range []string{"Endpoint", "AccessKey", "SecretKey", "Bucket"} {
			assert.Contains(t, err.Error(), field)
		}
	})
}

func TestNormalizePrefix(t *testing.T) {
	assert.Equal(t, "", normalizePrefix(""))
	assert.Equal(t, "staging/", normalizePrefix("staging"))
	assert.Equal(t, "staging/", normalizePrefix("staging/"))
}

func TestApplyPutOptions(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		applied := applyPutOptions(nil)
		assert.Empty(t, applied.contentType)
		assert.Nil(t, applied.metadata)
	})

	t.Run("last content type wins", func(t *testing.T) {
		applied := applyPutOptions([]PutOption{
			WithContentType("text/plain"),
			WithContentType("application/json"),
		})
		assert.Equal(t, "application/json", applied.contentType)
	})

	t.Run("nil options are skipped", func(t *testing.T) {
		applied := applyPutOptions([]PutOption{nil, WithContentType("text/plain"), nil})
		assert.Equal(t, "text/plain", applied.contentType)
	})
}

func baseConfig(fake *fakeS3) Config {
	return Config{
		Endpoint:  strings.TrimPrefix(fake.server.URL, "http://"),
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Bucket:    "bucket",
		Region:    "us-east-1",
	}
}

func TestNewMinIO(t *testing.T) {
	ctx := context.Background()

	t.Run("connects to an existing bucket", func(t *testing.T) {
		fake := newFakeS3(t)

		s, err := NewMinIO(ctx, baseConfig(fake))
		require.NoError(t, err)
		assert.NoError(t, s.Ping(ctx))
	})

	t.Run("missing bucket is an error by default", func(t *testing.T) {
		fake := newFakeS3(t)
		fake.dropBucket("bucket")

		_, err := NewMinIO(ctx, baseConfig(fake))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
		assert.Contains(t, err.Error(), "EnsureBucket", "the message should say how to fix it")
	})

	t.Run("EnsureBucket creates a missing bucket", func(t *testing.T) {
		fake := newFakeS3(t)
		fake.dropBucket("bucket")

		cfg := baseConfig(fake)
		cfg.EnsureBucket = true

		_, err := NewMinIO(ctx, cfg)
		require.NoError(t, err)
		assert.True(t, fake.hasBucket("bucket"))
	})

	t.Run("rejects incomplete config before connecting", func(t *testing.T) {
		_, err := NewMinIO(ctx, Config{Endpoint: "minio:9000"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required config")
	})

	t.Run("unreachable endpoint fails fast", func(t *testing.T) {
		cfg := Config{
			Endpoint:  "127.0.0.1:1",
			AccessKey: "k",
			SecretKey: "s",
			Bucket:    "bucket",
			Region:    "us-east-1",
		}

		_, err := NewMinIO(ctx, cfg)
		assert.Error(t, err)
	})

	t.Run("public endpoint is only used for signing", func(t *testing.T) {
		fake := newFakeS3(t)
		cfg := baseConfig(fake)
		cfg.PublicEndpoint = "cdn.example.com"
		cfg.PublicUseSSL = true

		s, err := NewMinIO(ctx, cfg)
		require.NoError(t, err)

		raw, err := s.PresignPut(ctx, "uploads/a.bin", time.Hour)
		require.NoError(t, err)

		parsed, err := url.Parse(raw)
		require.NoError(t, err)
		assert.Equal(t, "https", parsed.Scheme)
		assert.Equal(t, "cdn.example.com", parsed.Host)
	})
}
