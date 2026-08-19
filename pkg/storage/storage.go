// Package storage is an object storage abstraction over any S3-compatible backend
package storage

import (
	"context"
	"errors"
	"io"
	"iter"
	"time"
)

var ErrObjectNotFound = errors.New("storage: object not found")

// SizeUnknown tells Put the stream length is unknown
const SizeUnknown int64 = -1

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time

	Metadata map[string]string

	IsDir bool
}

// Storage is a bucket of objects addressed by key. Implementations must be safe
// for concurrent use.
type Storage interface {
	// Put stores the contents of r at key. size must be exact or SizeUnknown.
	Put(ctx context.Context, key string, r io.Reader, size int64, opts ...PutOption) error

	// Get opens the object at key; the caller must close the returned reader.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Stat returns metadata for key without transferring its contents.
	Stat(ctx context.Context, key string) (ObjectInfo, error)

	Exists(ctx context.Context, key string) (bool, error)

	// List walks the objects under prefix as a lazy sequence, yielding an error at
	// most once.
	List(ctx context.Context, prefix string, opts ...ListOption) iter.Seq2[ObjectInfo, error]

	// Copy duplicates srcKey to dstKey without moving bytes through this process.
	Copy(ctx context.Context, dstKey, srcKey string) error

	// Move copies then deletes; a crash in the middle leaves both keys, never
	// neither.
	Move(ctx context.Context, dstKey, srcKey string) error

	Delete(ctx context.Context, keys ...string) error

	// PresignGet returns a signed URL granting read access until it expires.
	PresignGet(ctx context.Context, key string, expiry time.Duration, opts ...PresignGetOption) (string, error)

	// PresignPut returns a signed URL to PUT to. It cannot bound the upload size.
	PresignPut(ctx context.Context, key string, expiry time.Duration, opts ...PresignPutOption) (string, error)

	// PresignPost returns a browser form upload whose size and content type the
	// object store enforces.
	PresignPost(ctx context.Context, key string, expiry time.Duration, opts ...PresignPostOption) (PostUpload, error)

	Ping(ctx context.Context) error
}

// PutOption customizes a single Put.
type PutOption func(*putOptions)

type putOptions struct {
	contentType        string
	contentEncoding    string
	cacheControl       string
	contentDisposition string
	metadata           map[string]string
}

// WithContentType sets the object's MIME type.
func WithContentType(contentType string) PutOption {
	return func(o *putOptions) { o.contentType = contentType }
}

// WithContentEncoding records how the bytes are encoded.
func WithContentEncoding(encoding string) PutOption {
	return func(o *putOptions) { o.contentEncoding = encoding }
}

// WithCacheControl sets the Cache-Control header served with the object.
func WithCacheControl(value string) PutOption {
	return func(o *putOptions) { o.cacheControl = value }
}

// WithContentDisposition sets the Content-Disposition header served with the
// object.
func WithContentDisposition(value string) PutOption {
	return func(o *putOptions) { o.contentDisposition = value }
}

// WithMetadata attaches user-defined metadata, returned by Stat.
func WithMetadata(metadata map[string]string) PutOption {
	return func(o *putOptions) { o.metadata = metadata }
}

func applyPutOptions(opts []PutOption) putOptions {
	var applied putOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&applied)
		}
	}
	return applied
}

// ListOption customizes a single List.
type ListOption func(*listOptions)

type listOptions struct {
	flat  bool
	limit int
}

// WithFlatHierarchy lists only the immediate children; sub-folders come back
// with IsDir set.
func WithFlatHierarchy() ListOption {
	return func(o *listOptions) { o.flat = true }
}

// WithLimit stops the listing after n objects.
func WithLimit(n int) ListOption {
	return func(o *listOptions) { o.limit = n }
}

func applyListOptions(opts []ListOption) listOptions {
	var applied listOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&applied)
		}
	}
	return applied
}

// PresignGetOption customizes a single PresignGet, overriding response headers
// for one URL.
type PresignGetOption func(*presignGetOptions)

type presignGetOptions struct {
	responseContentType        string
	responseContentDisposition string
}

// AsDownload makes the browser save the file under filename rather than render
// it.
func AsDownload(filename string) PresignGetOption {
	return func(o *presignGetOptions) {
		o.responseContentDisposition = `attachment; filename="` + sanitizeFilename(filename) + `"`
	}
}

// WithResponseContentType overrides the Content-Type served with this URL.
func WithResponseContentType(contentType string) PresignGetOption {
	return func(o *presignGetOptions) { o.responseContentType = contentType }
}

func applyPresignGetOptions(opts []PresignGetOption) presignGetOptions {
	var applied presignGetOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&applied)
		}
	}
	return applied
}

// PostUpload is a signed browser form upload produced by PresignPost.
type PostUpload struct {
	URL string

	Fields map[string]string
}

// PresignPostOption customizes a single PresignPost.
type PresignPostOption func(*presignPostOptions)

type presignPostOptions struct {
	minSize            int64
	maxSize            int64
	contentType        string
	contentTypePrefix  string
	contentDisposition string
	metadata           map[string]string
}

// WithMaxSize caps the upload; the object store rejects anything larger.
func WithMaxSize(bytes int64) PresignPostOption {
	return func(o *presignPostOptions) { o.maxSize = bytes }
}

// WithMinSize rejects uploads smaller than bytes.
func WithMinSize(bytes int64) PresignPostOption {
	return func(o *presignPostOptions) { o.minSize = bytes }
}

// WithAllowedContentType requires the upload to declare exactly this type.
func WithAllowedContentType(contentType string) PresignPostOption {
	return func(o *presignPostOptions) { o.contentType = contentType }
}

// WithAllowedContentTypePrefix requires the declared type to start with prefix.
func WithAllowedContentTypePrefix(prefix string) PresignPostOption {
	return func(o *presignPostOptions) { o.contentTypePrefix = prefix }
}

// WithPostContentDisposition sets the Content-Disposition stored with the uploaded object.
func WithPostContentDisposition(value string) PresignPostOption {
	return func(o *presignPostOptions) { o.contentDisposition = value }
}

// WithPostMetadata attaches user-defined metadata to the uploaded object.
func WithPostMetadata(metadata map[string]string) PresignPostOption {
	return func(o *presignPostOptions) { o.metadata = metadata }
}

func applyPresignPostOptions(opts []PresignPostOption) presignPostOptions {
	var applied presignPostOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&applied)
		}
	}
	return applied
}

// PresignPutOption customizes a single PresignPut.
type PresignPutOption func(*presignPutOptions)

type presignPutOptions struct {
	contentType string
}

// RequireContentType binds a Content-Type into the signature
func RequireContentType(contentType string) PresignPutOption {
	return func(o *presignPutOptions) { o.contentType = contentType }
}

func applyPresignPutOptions(opts []PresignPutOption) presignPutOptions {
	var applied presignPutOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&applied)
		}
	}
	return applied
}
