package storage

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func collect(t *testing.T, s Storage, prefix string, opts ...ListOption) []ObjectInfo {
	t.Helper()

	var objects []ObjectInfo
	for object, err := range s.List(context.Background(), prefix, opts...) {
		require.NoError(t, err)
		objects = append(objects, object)
	}
	return objects
}

func keysOf(objects []ObjectInfo) []string {
	keys := make([]string, 0, len(objects))
	for _, object := range objects {
		keys = append(keys, object.Key)
	}
	return keys
}

func TestMinIOStorage_List(t *testing.T) {
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})

	fake.seed("bucket/avatars/1.png", fakeObject{body: "a"})
	fake.seed("bucket/avatars/2.png", fakeObject{body: "bb"})
	fake.seed("bucket/avatars/thumbs/1.png", fakeObject{body: "ccc"})
	fake.seed("bucket/documents/report.pdf", fakeObject{body: "dddd"})

	t.Run("walks the whole subtree", func(t *testing.T) {
		got := collect(t, s, "avatars/")
		assert.Equal(t, []string{"avatars/1.png", "avatars/2.png", "avatars/thumbs/1.png"}, keysOf(got))
	})

	t.Run("reports size and modification time", func(t *testing.T) {
		got := collect(t, s, "avatars/1.png")
		require.Len(t, got, 1)
		assert.EqualValues(t, 1, got[0].Size)
		assert.False(t, got[0].LastModified.IsZero())
	})

	t.Run("flat hierarchy reports folders instead of descending", func(t *testing.T) {
		got := collect(t, s, "avatars/", WithFlatHierarchy())
		assert.Equal(t, []string{"avatars/1.png", "avatars/2.png", "avatars/thumbs/"}, keysOf(got))

		byKey := map[string]ObjectInfo{}
		for _, object := range got {
			byKey[object.Key] = object
		}
		assert.False(t, byKey["avatars/1.png"].IsDir)
		assert.True(t, byKey["avatars/thumbs/"].IsDir,
			"the sub-folder stands in for everything beneath it")
	})

	t.Run("a recursive listing has no folders", func(t *testing.T) {
		for _, object := range collect(t, s, "avatars/") {
			assert.False(t, object.IsDir, "%s", object.Key)
		}
	})

	t.Run("limit stops the walk", func(t *testing.T) {
		got := collect(t, s, "", WithLimit(2))
		assert.Len(t, got, 2)
	})

	t.Run("empty prefix walks the bucket", func(t *testing.T) {
		assert.Len(t, collect(t, s, ""), 4)
	})

	t.Run("no matches yields nothing", func(t *testing.T) {
		assert.Empty(t, collect(t, s, "nothing-here/"))
	})
}

func TestMinIOStorage_ListStopsWhenTheCallerBreaks(t *testing.T) {
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	for _, key := range []string{"a", "b", "c", "d"} {
		fake.seed("bucket/"+key, fakeObject{body: key})
	}

	seen := 0
	for _, err := range s.List(context.Background(), "") {
		require.NoError(t, err)
		seen++
		break
	}
	assert.Equal(t, 1, seen)
}

func TestMinIOStorage_ListAppliesKeyPrefix(t *testing.T) {
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{KeyPrefix: "staging"})

	fake.seed("bucket/staging/avatars/1.png", fakeObject{body: "a"})
	fake.seed("bucket/production/avatars/9.png", fakeObject{body: "b"})

	got := collect(t, s, "avatars/")
	assert.Equal(t, []string{"avatars/1.png"}, keysOf(got),
		"the prefix scopes the listing and is stripped from the results")
}

func TestMinIOStorage_Exists(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	fake.seed("bucket/present.txt", fakeObject{body: "x"})

	present, err := s.Exists(ctx, "present.txt")
	require.NoError(t, err)
	assert.True(t, present)

	absent, err := s.Exists(ctx, "absent.txt")
	require.NoError(t, err)
	assert.False(t, absent, "a missing object is not an error here")
}

func TestMinIOStorage_Copy(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	fake.seed("bucket/uploads/tmp-123", fakeObject{body: "payload", contentType: "image/png"})

	require.NoError(t, s.Copy(ctx, "avatars/7.png", "uploads/tmp-123"))

	assert.Equal(t, [][2]string{{"bucket/avatars/7.png", "bucket/uploads/tmp-123"}}, fake.recordedCopies())
	assert.True(t, fake.hasObject("bucket/avatars/7.png"))
	assert.True(t, fake.hasObject("bucket/uploads/tmp-123"), "copy must not consume the source")
}

func TestMinIOStorage_CopyMissingSource(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage(t, newFakeS3(t), Config{})

	assert.ErrorIs(t, s.Copy(ctx, "dst", "absent"), ErrObjectNotFound)
}

func TestMinIOStorage_DeleteBatch(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	for _, key := range []string{"a.txt", "b.txt", "c.txt"} {
		fake.seed("bucket/"+key, fakeObject{body: "x"})
	}

	require.NoError(t, s.Delete(ctx, "a.txt", "b.txt", "c.txt"))

	assert.ElementsMatch(t,
		[]string{"bucket/a.txt", "bucket/b.txt", "bucket/c.txt"},
		fake.deletedPaths(),
		"several keys should go out in one batched request")
	for _, key := range []string{"a.txt", "b.txt", "c.txt"} {
		assert.False(t, fake.hasObject("bucket/"+key))
	}
}

func TestMinIOStorage_DeleteNothing(t *testing.T) {
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})

	require.NoError(t, s.Delete(context.Background()))
	assert.Empty(t, fake.deletedPaths())
}

func TestMinIOStorage_StatReturnsMetadata(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	fake.seed("bucket/a.txt", fakeObject{
		body:        "x",
		contentType: "text/plain",
		metadata:    map[string]string{"owner": "kirk", "source": "upload"},
	})

	info, err := s.Stat(ctx, "a.txt")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"owner": "kirk", "source": "upload"}, info.Metadata,
		"metadata set on Put must be readable back")
}

func TestMinIOStorage_PutHeaders(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})

	require.NoError(t, s.Put(ctx, "asset.js", strings.NewReader("x"), 1,
		WithContentType("application/javascript"),
		WithContentEncoding("gzip"),
		WithCacheControl("public, max-age=31536000, immutable"),
		WithContentDisposition(`attachment; filename="asset.js"`),
		WithMetadata(map[string]string{"build": "abc123"}),
	))

	puts := fake.recordedPuts()
	require.Len(t, puts, 1)
	assert.Equal(t, "application/javascript", puts[0].contentType)
	assert.Equal(t, "gzip", puts[0].contentEncoding)
	assert.Equal(t, "public, max-age=31536000, immutable", puts[0].cacheControl)
	assert.Equal(t, `attachment; filename="asset.js"`, puts[0].contentDisposition)
	assert.Equal(t, "abc123", puts[0].metadata["build"])
}

func TestMinIOStorage_PresignGetOptions(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	fake.seed("bucket/report.pdf", fakeObject{body: "x", contentType: "application/pdf"})

	t.Run("as download", func(t *testing.T) {
		raw, err := s.PresignGet(ctx, "report.pdf", time.Hour, AsDownload("Q3 report.pdf"))
		require.NoError(t, err)

		parsed, err := url.Parse(raw)
		require.NoError(t, err)
		assert.Equal(t, `attachment; filename="Q3 report.pdf"`,
			parsed.Query().Get("response-content-disposition"))
	})

	t.Run("content type override", func(t *testing.T) {
		raw, err := s.PresignGet(ctx, "report.pdf", time.Hour, WithResponseContentType("text/plain"))
		require.NoError(t, err)

		parsed, err := url.Parse(raw)
		require.NoError(t, err)
		assert.Equal(t, "text/plain", parsed.Query().Get("response-content-type"))
	})

	t.Run("overrides are signed", func(t *testing.T) {
		plain, err := s.PresignGet(ctx, "report.pdf", time.Hour)
		require.NoError(t, err)
		decorated, err := s.PresignGet(ctx, "report.pdf", time.Hour, AsDownload("a.pdf"))
		require.NoError(t, err)

		plainSig, err := url.Parse(plain)
		require.NoError(t, err)
		decoratedSig, err := url.Parse(decorated)
		require.NoError(t, err)

		assert.NotEqual(t,
			plainSig.Query().Get("X-Amz-Signature"),
			decoratedSig.Query().Get("X-Amz-Signature"))
	})
}

func TestMinIOStorage_PresignPutRequireContentType(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage(t, newFakeS3(t), Config{})

	raw, err := s.PresignPut(ctx, "uploads/a.png", time.Hour, RequireContentType("image/png"))
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Contains(t, parsed.Query().Get("X-Amz-SignedHeaders"), "content-type",
		"the bound header must be part of the signature")
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "report.pdf", want: "report.pdf"},
		{name: "spaces are fine", input: "Q3 report.pdf", want: "Q3 report.pdf"},
		{name: "quotes would end the header value", input: `a".pdf`, want: "a.pdf"},
		{name: "path separators", input: "../../etc/passwd", want: "....etcpasswd"},
		{name: "header injection", input: "a.pdf\r\nX-Evil: 1", want: "a.pdfX-Evil: 1"},
		{name: "empty falls back", input: "", want: "download"},
		{name: "all stripped falls back", input: `"""`, want: "download"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeFilename(tt.input))
		})
	}
}

func TestApplyListOptions(t *testing.T) {
	applied := applyListOptions(nil)
	assert.False(t, applied.flat)
	assert.Zero(t, applied.limit)

	applied = applyListOptions([]ListOption{nil, WithFlatHierarchy(), WithLimit(10), nil})
	assert.True(t, applied.flat)
	assert.Equal(t, 10, applied.limit)
}

func TestMinIOStorage_Move(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})
	fake.seed("bucket/uploads/tmp-123", fakeObject{body: "payload"})

	require.NoError(t, s.Move(ctx, "avatars/7.png", "uploads/tmp-123"))

	assert.True(t, fake.hasObject("bucket/avatars/7.png"))
	assert.False(t, fake.hasObject("bucket/uploads/tmp-123"), "the source is removed")
	assert.Equal(t, []string{"bucket/uploads/tmp-123"}, fake.deletedPaths())
}

func TestMinIOStorage_MoveMissingSourceLeavesNothingBehind(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})

	assert.ErrorIs(t, s.Move(ctx, "dst", "absent"), ErrObjectNotFound)
	assert.Empty(t, fake.deletedPaths(), "nothing should have been deleted")
}

func TestMinIOStorage_PresignPost(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3(t)
	s := newFakeStorage(t, fake, Config{})

	t.Run("returns a form the browser can post", func(t *testing.T) {
		upload, err := s.PresignPost(ctx, "avatars/7.png", 15*time.Minute,
			WithMaxSize(5<<20),
			WithAllowedContentType("image/png"),
		)
		require.NoError(t, err)

		assert.NotEmpty(t, upload.URL)
		assert.Equal(t, "avatars/7.png", upload.Fields["key"])
		assert.Contains(t, upload.Fields, "policy", "the signed policy travels as a form field")
		assert.Contains(t, upload.Fields, "x-amz-signature")
		assert.Equal(t, "image/png", upload.Fields["Content-Type"])
	})

	t.Run("the key prefix is applied", func(t *testing.T) {
		scoped := newFakeStorage(t, fake, Config{KeyPrefix: "staging"})

		upload, err := scoped.PresignPost(ctx, "avatars/7.png", 15*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, "staging/avatars/7.png", upload.Fields["key"])
	})

	t.Run("a content type prefix accepts a family", func(t *testing.T) {
		upload, err := s.PresignPost(ctx, "a.bin", 15*time.Minute,
			WithAllowedContentTypePrefix("image/"))
		require.NoError(t, err)
		assert.NotEmpty(t, upload.Fields["policy"])
	})

	t.Run("metadata rides along", func(t *testing.T) {
		upload, err := s.PresignPost(ctx, "a.bin", 15*time.Minute,
			WithPostMetadata(map[string]string{"owner": "kirk"}))
		require.NoError(t, err)
		assert.Equal(t, "kirk", upload.Fields["x-amz-meta-owner"])
	})

	t.Run("expiry bounds are enforced", func(t *testing.T) {
		_, err := s.PresignPost(ctx, "a.bin", 0)
		assert.Error(t, err)

		_, err = s.PresignPost(ctx, "a.bin", presignExpiryLimit+time.Second)
		assert.Error(t, err)
	})

	t.Run("an impossible size range is rejected", func(t *testing.T) {
		_, err := s.PresignPost(ctx, "a.bin", time.Hour, WithMinSize(10), WithMaxSize(5))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "min size")
	})
}

func TestApplyPresignPostOptions(t *testing.T) {
	applied := applyPresignPostOptions(nil)
	assert.Zero(t, applied.maxSize)
	assert.Empty(t, applied.contentType)

	applied = applyPresignPostOptions([]PresignPostOption{
		nil,
		WithMaxSize(1024),
		WithMinSize(1),
		WithAllowedContentType("image/png"),
		WithPostContentDisposition("inline"),
	})
	assert.EqualValues(t, 1024, applied.maxSize)
	assert.EqualValues(t, 1, applied.minSize)
	assert.Equal(t, "image/png", applied.contentType)
	assert.Equal(t, "inline", applied.contentDisposition)
}
