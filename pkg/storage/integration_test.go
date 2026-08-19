package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func integrationStorage(t *testing.T) Storage {
	t.Helper()

	endpoint := os.Getenv("STORAGE_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("set STORAGE_TEST_ENDPOINT to run storage integration tests")
	}

	s, err := NewMinIO(context.Background(), Config{
		Endpoint:     endpoint,
		AccessKey:    os.Getenv("STORAGE_TEST_ACCESS_KEY"),
		SecretKey:    os.Getenv("STORAGE_TEST_SECRET_KEY"),
		Bucket:       "boot-integration",
		Region:       "us-east-1",
		EnsureBucket: true,
	})
	require.NoError(t, err)

	return s
}

func TestIntegration_ObjectLifecycle(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	key := "lifecycle/notes.txt"
	body := "the quick brown fox"
	t.Cleanup(func() { _ = s.Delete(ctx, key) })

	require.NoError(t, s.Put(ctx, key, strings.NewReader(body), int64(len(body)),
		WithContentType("text/plain"),
		WithMetadata(map[string]string{"owner": "kirk"}),
	))

	info, err := s.Stat(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, key, info.Key)
	assert.EqualValues(t, len(body), info.Size)
	assert.Equal(t, "text/plain", info.ContentType)
	assert.NotEmpty(t, info.ETag)
	assert.WithinDuration(t, time.Now(), info.LastModified, time.Minute)

	reader, err := s.Get(ctx, key)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	downloaded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, body, string(downloaded))

	require.NoError(t, s.Delete(ctx, key))
	_, err = s.Stat(ctx, key)
	assert.ErrorIs(t, err, ErrObjectNotFound)

	assert.NoError(t, s.Delete(ctx, key), "deleting twice must stay idempotent")
}

func TestIntegration_PutUnknownSize(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	key := "streamed/blob.bin"
	body := bytes.Repeat([]byte("x"), 1024)
	t.Cleanup(func() { _ = s.Delete(ctx, key) })

	require.NoError(t, s.Put(ctx, key, bytes.NewReader(body), SizeUnknown))

	info, err := s.Stat(ctx, key)
	require.NoError(t, err)
	assert.EqualValues(t, len(body), info.Size)
}

func TestIntegration_MissingObject(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	_, err := s.Stat(ctx, "definitely/not/here")
	assert.ErrorIs(t, err, ErrObjectNotFound)

	_, err = s.Get(ctx, "definitely/not/here")
	assert.ErrorIs(t, err, ErrObjectNotFound)

	_, err = s.PresignGet(ctx, "definitely/not/here", time.Minute)
	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestIntegration_PresignGetIsUsable(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	key := "presigned/report.txt"
	body := "quarterly numbers"
	t.Cleanup(func() { _ = s.Delete(ctx, key) })
	require.NoError(t, s.Put(ctx, key, strings.NewReader(body), int64(len(body)),
		WithContentType("text/plain")))

	signed, err := s.PresignGet(ctx, key, 5*time.Minute)
	require.NoError(t, err)

	resp := fetch(t, http.MethodGet, signed, nil)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	downloaded, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, body, string(downloaded))
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
}

func TestIntegration_PresignPutAcceptsAnUpload(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	key := "presigned/upload.txt"
	body := "uploaded by the client"
	t.Cleanup(func() { _ = s.Delete(ctx, key) })

	signed, err := s.PresignPut(ctx, key, 5*time.Minute)
	require.NoError(t, err)

	resp := fetch(t, http.MethodPut, signed, strings.NewReader(body))
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	reader, err := s.Get(ctx, key)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	stored, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, body, string(stored))
}

func TestIntegration_KeyPrefixIsolatesObjects(t *testing.T) {
	ctx := context.Background()
	endpoint := os.Getenv("STORAGE_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("set STORAGE_TEST_ENDPOINT to run storage integration tests")
	}

	newScoped := func(prefix string) Storage {
		s, err := NewMinIO(ctx, Config{
			Endpoint:     endpoint,
			AccessKey:    os.Getenv("STORAGE_TEST_ACCESS_KEY"),
			SecretKey:    os.Getenv("STORAGE_TEST_SECRET_KEY"),
			Bucket:       "boot-integration",
			Region:       "us-east-1",
			KeyPrefix:    prefix,
			EnsureBucket: true,
		})
		require.NoError(t, err)
		return s
	}

	staging, production := newScoped("staging"), newScoped("production")
	key := "shared-name.txt"
	t.Cleanup(func() {
		_ = staging.Delete(ctx, key)
		_ = production.Delete(ctx, key)
	})

	require.NoError(t, staging.Put(ctx, key, strings.NewReader("staging"), 7))

	_, err := production.Stat(ctx, key)
	assert.ErrorIs(t, err, ErrObjectNotFound, "the same key under another prefix is a different object")
}

func fetch(t *testing.T, method, rawURL string, body io.Reader) *http.Response {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, fmt.Sprintf("%s %s", method, rawURL))

	return resp
}

func TestIntegration_ListCopyDelete(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	keys := []string{
		"walk/a.txt",
		"walk/b.txt",
		"walk/nested/c.txt",
	}
	t.Cleanup(func() { _ = s.Delete(ctx, append(keys, "walk/copy.txt")...) })

	for _, key := range keys {
		require.NoError(t, s.Put(ctx, key, strings.NewReader(key), int64(len(key))))
	}

	t.Run("recursive listing walks the subtree", func(t *testing.T) {
		var found []string
		for object, err := range s.List(ctx, "walk/") {
			require.NoError(t, err)
			found = append(found, object.Key)
			assert.False(t, object.IsDir)
		}
		assert.ElementsMatch(t, keys, found)
	})

	t.Run("flat listing reports the sub-folder", func(t *testing.T) {
		var files, folders []string
		for object, err := range s.List(ctx, "walk/", WithFlatHierarchy()) {
			require.NoError(t, err)
			if object.IsDir {
				folders = append(folders, object.Key)
				continue
			}
			files = append(files, object.Key)
		}
		assert.ElementsMatch(t, []string{"walk/a.txt", "walk/b.txt"}, files)
		assert.Equal(t, []string{"walk/nested/"}, folders)
	})

	t.Run("limit stops the walk", func(t *testing.T) {
		count := 0
		for _, err := range s.List(ctx, "walk/", WithLimit(2)) {
			require.NoError(t, err)
			count++
		}
		assert.Equal(t, 2, count)
	})

	t.Run("copy leaves the source in place", func(t *testing.T) {
		require.NoError(t, s.Copy(ctx, "walk/copy.txt", "walk/a.txt"))

		copied, err := s.Stat(ctx, "walk/copy.txt")
		require.NoError(t, err)
		original, err := s.Stat(ctx, "walk/a.txt")
		require.NoError(t, err)
		assert.Equal(t, original.Size, copied.Size)
	})

	t.Run("batch delete removes everything in one request", func(t *testing.T) {
		require.NoError(t, s.Delete(ctx, keys...))
		for _, key := range keys {
			present, err := s.Exists(ctx, key)
			require.NoError(t, err)
			assert.False(t, present, "%s", key)
		}
	})
}

func TestIntegration_MetadataRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	key := "metadata/a.txt"
	t.Cleanup(func() { _ = s.Delete(ctx, key) })

	require.NoError(t, s.Put(ctx, key, strings.NewReader("x"), 1,
		WithContentType("text/plain"),
		WithCacheControl("public, max-age=60"),
		WithMetadata(map[string]string{"owner": "kirk"}),
	))

	info, err := s.Stat(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, "text/plain", info.ContentType)
	assert.Equal(t, "kirk", info.Metadata["owner"], "user metadata must survive the round trip")
}

func TestIntegration_PresignGetAsDownload(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	key := "presigned/inline.txt"
	t.Cleanup(func() { _ = s.Delete(ctx, key) })
	require.NoError(t, s.Put(ctx, key, strings.NewReader("body"), 4, WithContentType("text/plain")))

	signed, err := s.PresignGet(ctx, key, 5*time.Minute, AsDownload("quarterly report.txt"))
	require.NoError(t, err)

	resp := fetch(t, http.MethodGet, signed, nil)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, `attachment; filename="quarterly report.txt"`, resp.Header.Get("Content-Disposition"))
}

func TestIntegration_PresignPutEnforcesContentType(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	key := "presigned/typed.png"
	t.Cleanup(func() { _ = s.Delete(ctx, key) })

	signed, err := s.PresignPut(ctx, key, 5*time.Minute, RequireContentType("image/png"))
	require.NoError(t, err)

	t.Run("the declared type is accepted", func(t *testing.T) {
		req := newRequest(t, http.MethodPut, signed, strings.NewReader("fake png"))
		req.Header.Set("Content-Type", "image/png")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("anything else fails the signature", func(t *testing.T) {
		req := newRequest(t, http.MethodPut, signed, strings.NewReader("not a png"))
		req.Header.Set("Content-Type", "application/x-shellscript")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.GreaterOrEqual(t, resp.StatusCode, 400, "the store must refuse the mismatched upload")
	})
}

func newRequest(t *testing.T, method, rawURL string, body io.Reader) *http.Request {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	require.NoError(t, err)
	return req
}

func TestIntegration_PresignPostEnforcesSizeAndType(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	key := "post-policy/avatar.png"
	t.Cleanup(func() { _ = s.Delete(ctx, key) })

	upload, err := s.PresignPost(ctx, key, 15*time.Minute,
		WithMaxSize(1024),
		WithAllowedContentType("image/png"),
	)
	require.NoError(t, err)

	t.Run("a compliant upload is accepted", func(t *testing.T) {
		resp := postForm(t, upload, "avatar.png", "image/png", strings.Repeat("x", 512))
		defer func() { _ = resp.Body.Close() }()

		require.Less(t, resp.StatusCode, 400, "the store should accept it")

		info, err := s.Stat(ctx, key)
		require.NoError(t, err)
		assert.EqualValues(t, 512, info.Size)
	})

	t.Run("an oversized upload is refused before it lands", func(t *testing.T) {
		resp := postForm(t, upload, "avatar.png", "image/png", strings.Repeat("x", 4096))
		defer func() { _ = resp.Body.Close() }()

		assert.GreaterOrEqual(t, resp.StatusCode, 400,
			"the size cap is enforced by the store, not by the application")
	})

	t.Run("a mismatched content type is refused", func(t *testing.T) {
		resp := postForm(t, upload, "avatar.png", "application/x-shellscript", "x")
		defer func() { _ = resp.Body.Close() }()

		assert.GreaterOrEqual(t, resp.StatusCode, 400)
	})
}

func TestIntegration_Move(t *testing.T) {
	ctx := context.Background()
	s := integrationStorage(t)

	src, dst := "move/tmp-123", "move/final.txt"
	t.Cleanup(func() { _ = s.Delete(ctx, src, dst) })

	require.NoError(t, s.Put(ctx, src, strings.NewReader("payload"), 7))
	require.NoError(t, s.Move(ctx, dst, src))

	moved, err := s.Exists(ctx, dst)
	require.NoError(t, err)
	assert.True(t, moved)

	original, err := s.Exists(ctx, src)
	require.NoError(t, err)
	assert.False(t, original, "the source is gone")
}

func postForm(t *testing.T, upload PostUpload, filename, contentType, body string) *http.Response {
	t.Helper()

	var buffer bytes.Buffer
	form := multipart.NewWriter(&buffer)

	for name, value := range upload.Fields {
		if name == "Content-Type" {
			continue
		}
		require.NoError(t, form.WriteField(name, value))
	}

	require.NoError(t, form.WriteField("Content-Type", contentType))

	part, err := form.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, form.Close())

	req := newRequest(t, http.MethodPost, upload.URL, &buffer)
	req.Header.Set("Content-Type", form.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}
