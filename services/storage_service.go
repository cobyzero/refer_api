package services

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Buckets used by Refer App (R2 bucket names, created in the Cloudflare
// dashboard with public access enabled).
const (
	BucketUploads  = "uploads"  // legacy files migrated as-is
	BucketProducts = "products" // POST /api/products/upload
	BucketAvatars  = "avatars"  // POST /api/users/upload
)

// StorageService stores uploads in Cloudflare R2 with transparent fallback
// to local disk (./uploads). Flutter contract is unchanged either way:
// same endpoints, same {imageUrl}/{photoUrl} JSON shape.
type StorageService interface {
	// Enabled reports whether R2 is configured.
	Enabled() bool
	// Save stores data under bucket/filename and returns its public URL.
	// baseURL is used only for the disk fallback (same format as before).
	Save(bucket, filename string, data []byte, contentType, baseURL string) (string, error)
}

type storageService struct {
	client    *s3.Client
	publicURL map[string]string // bucket -> public base URL (no trailing slash)
}

// r2PublicURL returns the configured public base URL for a bucket ("" if none).
func r2PublicURL(bucket string) string {
	switch bucket {
	case BucketUploads:
		return strings.TrimRight(os.Getenv("R2_PUBLIC_UPLOADS_URL"), "/")
	case BucketProducts:
		return strings.TrimRight(os.Getenv("R2_PUBLIC_PRODUCTS_URL"), "/")
	case BucketAvatars:
		return strings.TrimRight(os.Getenv("R2_PUBLIC_AVATARS_URL"), "/")
	default:
		return ""
	}
}

// NewStorageService builds the service from R2_* env (server-side only).
// Required: R2_ENDPOINT, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY.
// Buckets resolve their public URL from R2_PUBLIC_<BUCKET>_URL.
func NewStorageService() StorageService {
	endpoint := strings.TrimRight(os.Getenv("R2_ENDPOINT"), "/")
	keyID := os.Getenv("R2_ACCESS_KEY_ID")
	secret := os.Getenv("R2_SECRET_ACCESS_KEY")
	if endpoint == "" || keyID == "" || secret == "" {
		return &storageService{}
	}
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: &endpoint,
		Credentials:  credentials.NewStaticCredentialsProvider(keyID, secret, ""),
		UsePathStyle: true,
	})
	return &storageService{
		client: client,
		publicURL: map[string]string{
			BucketUploads:  r2PublicURL(BucketUploads),
			BucketProducts: r2PublicURL(BucketProducts),
			BucketAvatars:  r2PublicURL(BucketAvatars),
		},
	}
}

func (s *storageService) Enabled() bool {
	return s.client != nil
}

// Save tries R2 first; on any error it falls back to ./uploads disk (same
// naming + URL format as the legacy handlers) so uploads never break.
// Failures are logged server-side, never leaked.
func (s *storageService) Save(bucket, filename string, data []byte, contentType, baseURL string) (string, error) {
	if s.Enabled() {
		if url, err := s.upload(bucket, filename, data, contentType); err == nil {
			return url, nil
		} else {
			log.Printf("storage upload failed, falling back to disk: %v", err)
		}
	}
	return saveToDisk(filename, data, baseURL)
}

func (s *storageService) upload(bucket, filename string, data []byte, contentType string) (string, error) {
	base := s.publicURL[bucket]
	if base == "" {
		// Re-read env so Dokploy changes apply without rebuild.
		base = r2PublicURL(bucket)
		if base == "" {
			return "", fmt.Errorf("no public URL configured for bucket %q (R2_PUBLIC_<BUCKET>_URL)", bucket)
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	key := filepath.Base(filename)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("r2 upload (bucket %s): %w", bucket, err)
	}
	return fmt.Sprintf("%s/%s", base, key), nil
}

// SaveToDisk preserves the exact legacy disk behavior for callers without a
// wired StorageService (e.g. unit tests building handlers directly).
func SaveToDisk(filename string, data []byte, baseURL string) (string, error) {
	return saveToDisk(filename, data, baseURL)
}

// saveToDisk preserves the exact legacy behavior (naming is done by callers).
func saveToDisk(filename string, data []byte, baseURL string) (string, error) {
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}
	destPath := filepath.Join(uploadDir, filepath.Base(filename))
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}
	return fmt.Sprintf("%s/uploads/%s", strings.TrimRight(baseURL, "/"), filepath.Base(filename)), nil
}
