package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint string

	// PublicEndpoint is the host browsers reach the store on.
	PublicEndpoint string

	UseSSL       bool
	PublicUseSSL bool

	// AccessKey and SecretKey authenticate to the store. Required.
	AccessKey string
	SecretKey string

	// Region is the S3 region; empty is fine for MinIO.
	Region string

	// Bucket is the bucket this Storage reads and writes. Required.
	Bucket string

	// KeyPrefix is prepended to every key.
	KeyPrefix string

	// EnsureBucket creates the bucket at startup when it does not exist.
	EnsureBucket bool
}

const presignExpiryLimit = 7 * 24 * time.Hour

const userMetadataPrefix = "X-Amz-Meta-"

type minioStorage struct {
	client *minio.Client

	signer *minio.Client

	bucket string
	prefix string
}

func NewMinIO(ctx context.Context, cfg Config) (Storage, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	client, err := newMinIOClient(cfg.Endpoint, cfg.UseSSL, cfg)
	if err != nil {
		return nil, fmt.Errorf("storage: connect to %s: %w", cfg.Endpoint, err)
	}

	signer := client
	if cfg.PublicEndpoint != "" && cfg.PublicEndpoint != cfg.Endpoint {
		signer, err = newMinIOClient(cfg.PublicEndpoint, cfg.PublicUseSSL, cfg)
		if err != nil {
			return nil, fmt.Errorf("storage: build signer for %s: %w", cfg.PublicEndpoint, err)
		}
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage: check bucket %q: %w", cfg.Bucket, err)
	}
	if !exists {
		if !cfg.EnsureBucket {
			return nil, fmt.Errorf("storage: bucket %q does not exist (set EnsureBucket to create it)", cfg.Bucket)
		}
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("storage: create bucket %q: %w", cfg.Bucket, err)
		}
	}

	return &minioStorage{
		client: client,
		signer: signer,
		bucket: cfg.Bucket,
		prefix: normalizePrefix(cfg.KeyPrefix),
	}, nil
}

func newMinIOClient(endpoint string, useSSL bool, cfg Config) (*minio.Client, error) {
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: useSSL,
		Region: cfg.Region,
	})
}

func (cfg Config) validate() error {
	var missing []string
	if cfg.Endpoint == "" {
		missing = append(missing, "Endpoint")
	}
	if cfg.AccessKey == "" {
		missing = append(missing, "AccessKey")
	}
	if cfg.SecretKey == "" {
		missing = append(missing, "SecretKey")
	}
	if cfg.Bucket == "" {
		missing = append(missing, "Bucket")
	}
	if len(missing) > 0 {
		return fmt.Errorf("storage: missing required config: %s", strings.Join(missing, ", "))
	}
	return nil
}

func normalizePrefix(prefix string) string {
	if prefix == "" || strings.HasSuffix(prefix, "/") {
		return prefix
	}
	return prefix + "/"
}

func (s *minioStorage) key(key string) string { return s.prefix + key }

func (s *minioStorage) bareKey(key string) string {
	return strings.TrimPrefix(key, s.prefix)
}

func (s *minioStorage) Put(ctx context.Context, key string, r io.Reader, size int64, opts ...PutOption) error {
	applied := applyPutOptions(opts)

	_, err := s.client.PutObject(ctx, s.bucket, s.key(key), r, size, minio.PutObjectOptions{
		ContentType:        applied.contentType,
		ContentEncoding:    applied.contentEncoding,
		ContentDisposition: applied.contentDisposition,
		CacheControl:       applied.cacheControl,
		UserMetadata:       applied.metadata,
	})
	if err != nil {
		return fmt.Errorf("storage: put %q: %w", key, err)
	}
	return nil
}

func (s *minioStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucket, s.key(key), minio.GetObjectOptions{})
	if err != nil {
		return nil, translateError(err, key, "get")
	}

	if _, err := object.Stat(); err != nil {
		_ = object.Close()
		return nil, translateError(err, key, "get")
	}
	return object, nil
}

func (s *minioStorage) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucket, s.key(key), minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, translateError(err, key, "stat")
	}
	return s.toObjectInfo(info), nil
}

func (s *minioStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.Stat(ctx, key)
	if errors.Is(err, ErrObjectNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *minioStorage) List(ctx context.Context, prefix string, opts ...ListOption) iter.Seq2[ObjectInfo, error] {
	applied := applyListOptions(opts)

	return func(yield func(ObjectInfo, error) bool) {
		listing := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
			Prefix: s.key(prefix),

			Recursive: !applied.flat,
		})

		yielded := 0
		for object := range listing {
			if object.Err != nil {
				yield(ObjectInfo{}, fmt.Errorf("storage: list %q: %w", prefix, object.Err))
				return
			}
			entry := s.toObjectInfo(object)

			entry.IsDir = applied.flat && strings.HasSuffix(object.Key, "/")

			if !yield(entry, nil) {
				return
			}

			yielded++
			if applied.limit > 0 && yielded >= applied.limit {
				return
			}
		}
	}
}

func (s *minioStorage) Copy(ctx context.Context, dstKey, srcKey string) error {
	_, err := s.client.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: s.bucket, Object: s.key(dstKey)},
		minio.CopySrcOptions{Bucket: s.bucket, Object: s.key(srcKey)},
	)
	if err != nil {
		return translateError(err, srcKey, "copy")
	}
	return nil
}

func (s *minioStorage) Move(ctx context.Context, dstKey, srcKey string) error {
	if err := s.Copy(ctx, dstKey, srcKey); err != nil {
		return err
	}
	if err := s.Delete(ctx, srcKey); err != nil {
		return fmt.Errorf("storage: move %q to %q: copied but source not removed: %w", srcKey, dstKey, err)
	}
	return nil
}

func (s *minioStorage) Delete(ctx context.Context, keys ...string) error {
	switch len(keys) {
	case 0:
		return nil
	case 1:
		if err := s.client.RemoveObject(ctx, s.bucket, s.key(keys[0]), minio.RemoveObjectOptions{}); err != nil {
			return fmt.Errorf("storage: delete %q: %w", keys[0], err)
		}
		return nil
	}

	objects := func(yield func(minio.ObjectInfo) bool) {
		for _, key := range keys {
			if !yield(minio.ObjectInfo{Key: s.key(key)}) {
				return
			}
		}
	}

	results, err := s.client.RemoveObjectsWithIter(ctx, s.bucket, objects, minio.RemoveObjectsOptions{})
	if err != nil {
		return fmt.Errorf("storage: delete %d key(s): %w", len(keys), err)
	}
	for result := range results {
		if result.Err != nil {
			return fmt.Errorf("storage: delete %q: %w", s.bareKey(result.ObjectName), result.Err)
		}
	}
	return nil
}

func (s *minioStorage) PresignGet(
	ctx context.Context,
	key string,
	expiry time.Duration,
	opts ...PresignGetOption,
) (string, error) {
	if err := validateExpiry(expiry); err != nil {
		return "", err
	}

	if _, err := s.Stat(ctx, key); err != nil {
		return "", err
	}

	applied := applyPresignGetOptions(opts)
	params := url.Values{}
	if applied.responseContentType != "" {
		params.Set("response-content-type", applied.responseContentType)
	}
	if applied.responseContentDisposition != "" {
		params.Set("response-content-disposition", applied.responseContentDisposition)
	}

	signed, err := s.signer.PresignedGetObject(ctx, s.bucket, s.key(key), expiry, params)
	if err != nil {
		return "", fmt.Errorf("storage: presign get %q: %w", key, err)
	}
	return signed.String(), nil
}

func (s *minioStorage) PresignPut(
	ctx context.Context,
	key string,
	expiry time.Duration,
	opts ...PresignPutOption,
) (string, error) {
	if err := validateExpiry(expiry); err != nil {
		return "", err
	}
	applied := applyPresignPutOptions(opts)

	if applied.contentType == "" {
		signed, err := s.signer.PresignedPutObject(ctx, s.bucket, s.key(key), expiry)
		if err != nil {
			return "", fmt.Errorf("storage: presign put %q: %w", key, err)
		}
		return signed.String(), nil
	}

	headers := http.Header{}
	headers.Set("Content-Type", applied.contentType)

	signed, err := s.signer.PresignHeader(ctx, http.MethodPut, s.bucket, s.key(key), expiry, url.Values{}, headers)
	if err != nil {
		return "", fmt.Errorf("storage: presign put %q: %w", key, err)
	}
	return signed.String(), nil
}

func (s *minioStorage) PresignPost(
	ctx context.Context,
	key string,
	expiry time.Duration,
	opts ...PresignPostOption,
) (PostUpload, error) {
	if err := validateExpiry(expiry); err != nil {
		return PostUpload{}, err
	}
	applied := applyPresignPostOptions(opts)

	policy := minio.NewPostPolicy()
	if err := policy.SetBucket(s.bucket); err != nil {
		return PostUpload{}, fmt.Errorf("storage: presign post %q: %w", key, err)
	}
	if err := policy.SetKey(s.key(key)); err != nil {
		return PostUpload{}, fmt.Errorf("storage: presign post %q: %w", key, err)
	}

	if err := policy.SetExpires(time.Now().UTC().Add(expiry)); err != nil {
		return PostUpload{}, fmt.Errorf("storage: presign post %q: %w", key, err)
	}

	if applied.maxSize > 0 {
		if applied.minSize < 0 || applied.minSize > applied.maxSize {
			return PostUpload{}, fmt.Errorf(
				"storage: presign post %q: min size %d must be between 0 and max size %d",
				key, applied.minSize, applied.maxSize)
		}
		if err := policy.SetContentLengthRange(applied.minSize, applied.maxSize); err != nil {
			return PostUpload{}, fmt.Errorf("storage: presign post %q: %w", key, err)
		}
	}

	switch {
	case applied.contentType != "":
		if err := policy.SetContentType(applied.contentType); err != nil {
			return PostUpload{}, fmt.Errorf("storage: presign post %q: %w", key, err)
		}
	case applied.contentTypePrefix != "":
		if err := policy.SetContentTypeStartsWith(applied.contentTypePrefix); err != nil {
			return PostUpload{}, fmt.Errorf("storage: presign post %q: %w", key, err)
		}
	}

	if applied.contentDisposition != "" {
		if err := policy.SetContentDisposition(applied.contentDisposition); err != nil {
			return PostUpload{}, fmt.Errorf("storage: presign post %q: %w", key, err)
		}
	}
	for name, value := range applied.metadata {
		if err := policy.SetUserMetadata(name, value); err != nil {
			return PostUpload{}, fmt.Errorf("storage: presign post %q: %w", key, err)
		}
	}

	endpoint, fields, err := s.signer.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return PostUpload{}, fmt.Errorf("storage: presign post %q: %w", key, err)
	}
	return PostUpload{URL: endpoint.String(), Fields: fields}, nil
}

func (s *minioStorage) Ping(ctx context.Context) error {
	if _, err := s.client.BucketExists(ctx, s.bucket); err != nil {
		return fmt.Errorf("storage: ping bucket %q: %w", s.bucket, err)
	}
	return nil
}

func (s *minioStorage) toObjectInfo(info minio.ObjectInfo) ObjectInfo {
	return ObjectInfo{
		Key:          s.bareKey(info.Key),
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified,
		Metadata:     userMetadata(info.Metadata),
	}
}

func userMetadata(header http.Header) map[string]string {
	var metadata map[string]string
	for name, values := range header {
		suffix, found := strings.CutPrefix(http.CanonicalHeaderKey(name), userMetadataPrefix)
		if !found || len(values) == 0 {
			continue
		}
		if metadata == nil {
			metadata = make(map[string]string)
		}
		metadata[strings.ToLower(suffix)] = values[0]
	}
	return metadata
}

func validateExpiry(expiry time.Duration) error {
	if expiry <= 0 {
		return fmt.Errorf("storage: presign expiry must be positive, got %v", expiry)
	}
	if expiry > presignExpiryLimit {
		return fmt.Errorf("storage: presign expiry must be at most %v, got %v", presignExpiryLimit, expiry)
	}
	return nil
}

func sanitizeFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r < 0x20, r == 0x7f:
			return -1
		case r == '"', r == '\\', r == '/':
			return -1
		}
		return r
	}, name)

	if name == "" {
		return "download"
	}
	return name
}

func translateError(err error, key, op string) error {
	if err == nil {
		return nil
	}
	switch minio.ToErrorResponse(err).Code {
	case "NoSuchKey", "NoSuchObject":
		return fmt.Errorf("storage: %s %q: %w", op, key, ErrObjectNotFound)
	}
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("storage: %s %q: %w", op, key, ErrObjectNotFound)
	}
	return fmt.Errorf("storage: %s %q: %w", op, key, err)
}
