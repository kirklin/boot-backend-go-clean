package storage

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type fakeObject struct {
	body         string
	contentType  string
	etag         string
	lastModified time.Time
	metadata     map[string]string
}

type recordedPut struct {
	path               string
	contentType        string
	cacheControl       string
	contentDisposition string
	contentEncoding    string
	metadata           map[string]string
}

type fakeS3 struct {
	server *httptest.Server

	mu      sync.Mutex
	buckets map[string]bool
	objects map[string]fakeObject
	puts    []recordedPut
	deletes []string
	copies  [][2]string
}

func newFakeS3(t *testing.T) *fakeS3 {
	t.Helper()

	fake := &fakeS3{
		buckets: map[string]bool{"bucket": true},
		objects: make(map[string]fakeObject),
	}
	fake.server = httptest.NewServer(http.HandlerFunc(fake.handle))
	t.Cleanup(fake.server.Close)

	return fake
}

func (f *fakeS3) seed(path string, obj fakeObject) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if obj.etag == "" {
		obj.etag = "d41d8cd98f00b204e9800998ecf8427e"
	}
	if obj.lastModified.IsZero() {
		obj.lastModified = time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	}
	f.objects[path] = obj
}

func (f *fakeS3) dropBucket(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.buckets, name)
}

func (f *fakeS3) hasBucket(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buckets[name]
}

func (f *fakeS3) hasObject(path string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[path]
	return ok
}

func (f *fakeS3) recordedPuts() []recordedPut {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedPut(nil), f.puts...)
}

func (f *fakeS3) deletedPaths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.deletes...)
}

func (f *fakeS3) recordedCopies() [][2]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][2]string(nil), f.copies...)
}

func (f *fakeS3) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	if bucket, isRoot := strings.CutSuffix(path, "/"); isRoot {
		f.handleBucket(w, r, bucket)
		return
	}
	f.handleObject(w, r, path)
}

func (f *fakeS3) handleBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case r.Method == http.MethodHead:
		if !f.buckets[bucket] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

	case r.Method == http.MethodPut:
		f.buckets[bucket] = true
		w.WriteHeader(http.StatusOK)

	case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
		f.writeListing(w, bucket, r.URL.Query().Get("prefix"), r.URL.Query().Get("delimiter"))

	case r.Method == http.MethodPost && r.URL.Query().Has("delete"):
		f.writeBatchDelete(w, r, bucket)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (f *fakeS3) handleObject(w http.ResponseWriter, r *http.Request, path string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch r.Method {
	case http.MethodPut:
		if source := r.Header.Get("X-Amz-Copy-Source"); source != "" {
			f.writeCopy(w, path, strings.TrimPrefix(source, "/"))
			return
		}
		f.puts = append(f.puts, recordedPut{
			path:               path,
			contentType:        r.Header.Get("Content-Type"),
			cacheControl:       r.Header.Get("Cache-Control"),
			contentDisposition: r.Header.Get("Content-Disposition"),
			contentEncoding:    r.Header.Get("Content-Encoding"),
			metadata:           extractUserMetadata(r.Header),
		})
		w.Header().Set("ETag", `"fake-etag"`)
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		f.deletes = append(f.deletes, path)
		delete(f.objects, path)
		w.WriteHeader(http.StatusNoContent)

	case http.MethodHead, http.MethodGet:
		object, ok := f.objects[path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", object.contentType)
		w.Header().Set("ETag", `"`+object.etag+`"`)
		w.Header().Set("Last-Modified", object.lastModified.Format(http.TimeFormat))
		w.Header().Set("Content-Length", strconv.Itoa(len(object.body)))
		for name, value := range object.metadata {
			w.Header().Set(userMetadataPrefix+name, value)
		}
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(object.body))
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type listBucketResult struct {
	XMLName        xml.Name         `xml:"ListBucketResult"`
	Name           string           `xml:"Name"`
	Prefix         string           `xml:"Prefix"`
	KeyCount       int              `xml:"KeyCount"`
	MaxKeys        int              `xml:"MaxKeys"`
	IsTruncated    bool             `xml:"IsTruncated"`
	Contents       []listEntry      `xml:"Contents"`
	CommonPrefixes []commonPrefixes `xml:"CommonPrefixes"`
}

type listEntry struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

type commonPrefixes struct {
	Prefix string `xml:"Prefix"`
}

func (f *fakeS3) writeListing(w http.ResponseWriter, bucket, prefix, delimiter string) {
	result := listBucketResult{Name: bucket, Prefix: prefix, MaxKeys: 1000}
	folders := map[string]struct{}{}

	keys := make([]string, 0, len(f.objects))
	for path := range f.objects {
		keys = append(keys, path)
	}
	sort.Strings(keys)

	for _, path := range keys {
		key, inBucket := strings.CutPrefix(path, bucket+"/")
		if !inBucket || !strings.HasPrefix(key, prefix) {
			continue
		}

		if delimiter != "" {
			rest := strings.TrimPrefix(key, prefix)
			if index := strings.Index(rest, delimiter); index >= 0 {
				folders[prefix+rest[:index+len(delimiter)]] = struct{}{}
				continue
			}
		}

		object := f.objects[path]
		result.Contents = append(result.Contents, listEntry{
			Key:          key,
			LastModified: object.lastModified.UTC().Format(time.RFC3339),
			ETag:         `"` + object.etag + `"`,
			Size:         int64(len(object.body)),
			StorageClass: "STANDARD",
		})
	}

	for folder := range folders {
		result.CommonPrefixes = append(result.CommonPrefixes, commonPrefixes{Prefix: folder})
	}
	sort.Slice(result.CommonPrefixes, func(i, j int) bool {
		return result.CommonPrefixes[i].Prefix < result.CommonPrefixes[j].Prefix
	})
	result.KeyCount = len(result.Contents)

	writeXML(w, result)
}

type deleteRequest struct {
	XMLName xml.Name              `xml:"Delete"`
	Objects []deleteRequestObject `xml:"Object"`
}

type deleteRequestObject struct {
	Key string `xml:"Key"`
}

type deleteResult struct {
	XMLName xml.Name       `xml:"DeleteResult"`
	Deleted []deletedEntry `xml:"Deleted"`
}

type deletedEntry struct {
	Key string `xml:"Key"`
}

func (f *fakeS3) writeBatchDelete(w http.ResponseWriter, r *http.Request, bucket string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var request deleteRequest
	if err := xml.Unmarshal(body, &request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var result deleteResult
	for _, object := range request.Objects {
		path := bucket + "/" + object.Key
		f.deletes = append(f.deletes, path)
		delete(f.objects, path)
		result.Deleted = append(result.Deleted, deletedEntry(object))
	}

	writeXML(w, result)
}

type copyObjectResult struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	LastModified string   `xml:"LastModified"`
	ETag         string   `xml:"ETag"`
}

func (f *fakeS3) writeCopy(w http.ResponseWriter, dstPath, srcPath string) {
	source, ok := f.objects[srcPath]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	f.copies = append(f.copies, [2]string{dstPath, srcPath})
	f.objects[dstPath] = source

	writeXML(w, copyObjectResult{
		LastModified: source.lastModified.UTC().Format(time.RFC3339),
		ETag:         `"` + source.etag + `"`,
	})
}

func writeXML(w http.ResponseWriter, payload any) {
	encoded, err := xml.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(encoded)
}

func extractUserMetadata(header http.Header) map[string]string {
	metadata := make(map[string]string)
	for key, values := range header {
		if after, found := strings.CutPrefix(http.CanonicalHeaderKey(key), userMetadataPrefix); found && len(values) > 0 {
			metadata[strings.ToLower(after)] = values[0]
		}
	}
	return metadata
}

func newFakeStorage(t *testing.T, fake *fakeS3, cfg Config) *minioStorage {
	t.Helper()

	cfg.Endpoint = strings.TrimPrefix(fake.server.URL, "http://")
	cfg.AccessKey = "test-access-key"
	cfg.SecretKey = "test-secret-key"
	if cfg.Bucket == "" {
		cfg.Bucket = "bucket"
	}

	cfg.Region = "us-east-1"

	client, err := newMinIOClient(cfg.Endpoint, false, cfg)
	if err != nil {
		t.Fatalf("build client: %v", err)
	}

	signer := client
	if cfg.PublicEndpoint != "" {
		signer, err = minio.New(cfg.PublicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.PublicUseSSL,
			Region: cfg.Region,
		})
		if err != nil {
			t.Fatalf("build signer: %v", err)
		}
	}

	return &minioStorage{
		client: client,
		signer: signer,
		bucket: cfg.Bucket,
		prefix: normalizePrefix(cfg.KeyPrefix),
	}
}
