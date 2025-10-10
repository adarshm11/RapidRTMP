package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCSStorage implements Storage using Google Cloud Storage
type GCSStorage struct {
	client     *storage.Client
	bucketName string
	baseDir    string
	ctx        context.Context
}

// NewGCSStorage creates a new GCS storage instance
// projectID: Your GCP project ID
// bucketName: The GCS bucket name
// baseDir: Base directory/prefix within the bucket (e.g., "streams")
func NewGCSStorage(ctx context.Context, projectID, bucketName, baseDir string) (*GCSStorage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Verify bucket exists
	bucket := client.Bucket(bucketName)
	if _, err := bucket.Attrs(ctx); err != nil {
		return nil, fmt.Errorf("failed to access bucket %s: %w", bucketName, err)
	}

	return &GCSStorage{
		client:     client,
		bucketName: bucketName,
		baseDir:    baseDir,
		ctx:        ctx,
	}, nil
}

// Write writes data to GCS
func (s *GCSStorage) Write(path string, data []byte) error {
	objectPath := s.fullPath(path)
	
	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	w := obj.NewWriter(s.ctx)
	
	// Set metadata
	w.ContentType = s.getContentType(path)
	w.CacheControl = s.getCacheControl(path)
	
	// Write data
	if _, err := w.Write(data); err != nil {
		w.Close()
		return fmt.Errorf("failed to write to GCS: %w", err)
	}
	
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close GCS writer: %w", err)
	}
	
	return nil
}

// Read reads data from GCS
func (s *GCSStorage) Read(path string) ([]byte, error) {
	objectPath := s.fullPath(path)
	
	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	r, err := obj.NewReader(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read from GCS: %w", err)
	}
	defer r.Close()
	
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	
	return data, nil
}

// ReadSeeker returns a ReadSeeker for GCS object
func (s *GCSStorage) ReadSeeker(path string) (io.ReadSeeker, error) {
	objectPath := s.fullPath(path)
	
	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	
	// For GCS, we need to wrap the reader to support seeking
	// This is a simplified implementation - for production, consider using
	// signed URLs or byte-range requests
	r, err := obj.NewReader(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open GCS object: %w", err)
	}
	
	// Read all data into memory (for seeking support)
	// For large files, consider implementing a custom seeker with byte-range requests
	data, err := io.ReadAll(r)
	r.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read GCS object: %w", err)
	}
	
	return &bytesReadSeeker{data: data}, nil
}

// Delete deletes a file from GCS
func (s *GCSStorage) Delete(path string) error {
	objectPath := s.fullPath(path)
	
	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	if err := obj.Delete(s.ctx); err != nil && err != storage.ErrObjectNotExist {
		return fmt.Errorf("failed to delete from GCS: %w", err)
	}
	
	return nil
}

// Exists checks if a file exists in GCS
func (s *GCSStorage) Exists(path string) (bool, error) {
	objectPath := s.fullPath(path)
	
	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	_, err := obj.Attrs(s.ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check GCS object: %w", err)
	}
	
	return true, nil
}

// List lists files in a directory in GCS
func (s *GCSStorage) List(dir string) ([]string, error) {
	prefix := s.fullPath(dir)
	if prefix != "" && prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	
	query := &storage.Query{
		Prefix: prefix,
	}
	
	it := s.client.Bucket(s.bucketName).Objects(s.ctx, query)
	
	var files []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list GCS objects: %w", err)
		}
		
		// Extract filename from full path
		name := attrs.Name
		if len(name) > len(prefix) {
			name = name[len(prefix):]
		}
		
		// Skip directories (objects ending with /)
		if name != "" && name[len(name)-1] != '/' {
			files = append(files, name)
		}
	}
	
	return files, nil
}

// Close closes the GCS client
func (s *GCSStorage) Close() error {
	return s.client.Close()
}

// GetSignedURL generates a signed URL for public access
func (s *GCSStorage) GetSignedURL(path string, expiration time.Duration) (string, error) {
	objectPath := s.fullPath(path)
	
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiration),
	}
	
	url, err := s.client.Bucket(s.bucketName).SignedURL(objectPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}
	
	return url, nil
}

// Helper functions

func (s *GCSStorage) fullPath(path string) string {
	if s.baseDir == "" {
		return path
	}
	return s.baseDir + "/" + path
}

func (s *GCSStorage) getContentType(path string) string {
	// Determine content type based on file extension
	if len(path) >= 5 && path[len(path)-5:] == ".m3u8" {
		return "application/vnd.apple.mpegurl"
	}
	if len(path) >= 4 && path[len(path)-4:] == ".m4s" {
		return "video/iso.segment"
	}
	if len(path) >= 4 && path[len(path)-4:] == ".mp4" {
		return "video/mp4"
	}
	return "application/octet-stream"
}

func (s *GCSStorage) getCacheControl(path string) string {
	// Playlists should not be cached (low latency)
	if len(path) >= 5 && path[len(path)-5:] == ".m3u8" {
		return "no-cache, no-store, must-revalidate"
	}
	// Segments and init files can be cached
	if len(path) >= 4 && (path[len(path)-4:] == ".m4s" || path[len(path)-4:] == ".mp4") {
		return "public, max-age=3600"
	}
	return "public, max-age=300"
}

// bytesReadSeeker implements io.ReadSeeker for in-memory data
type bytesReadSeeker struct {
	data []byte
	pos  int64
}

func (b *bytesReadSeeker) Read(p []byte) (n int, err error) {
	if b.pos >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += int64(n)
	return n, nil
}

func (b *bytesReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = b.pos + offset
	case io.SeekEnd:
		newPos = int64(len(b.data)) + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}
	
	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}
	
	b.pos = newPos
	return newPos, nil
}

