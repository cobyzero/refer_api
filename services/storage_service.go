package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Buckets used by Refer App (see supabase/storage.sql).
const (
	BucketUploads  = "uploads"  // legacy files migrated as-is
	BucketProducts = "products" // POST /api/products/upload
	BucketAvatars  = "avatars"  // POST /api/users/upload
)

// StorageService stores uploads in Supabase Storage with transparent fallback
// to local disk (./uploads). Flutter contract is unchanged either way:
// same endpoints, same {imageUrl}/{photoUrl} JSON shape.
type StorageService interface {
	// Enabled reports whether Supabase Storage is configured.
	Enabled() bool
	// Save stores data under bucket/filename and returns its public URL.
	// baseURL is used only for the disk fallback (same format as before).
	Save(bucket, filename string, data []byte, contentType, baseURL string) (string, error)
}

type storageService struct {
	baseURL    string // internal, e.g. http://supabase-kong:8000 (upload path)
	publicURL  string // public, e.g. http://localhost:8000 (URL returned to Flutter)
	serviceKey string
	anonKey    string
	httpClient *http.Client
}

// NewStorageService builds the service from SUPABASE_* env (server-side only).
func NewStorageService() StorageService {
	return &storageService{
		baseURL:    strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		publicURL:  strings.TrimRight(os.Getenv("SUPABASE_PUBLIC_URL"), "/"),
		serviceKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		anonKey:    os.Getenv("SUPABASE_ANON_KEY"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *storageService) Enabled() bool {
	return s.baseURL != "" && s.serviceKey != ""
}

func (s *storageService) publicURLFor(bucket, filename string) string {
	base := s.publicURL
	if base == "" {
		base = s.baseURL
	}
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", base, bucket, filename)
}

// Save tries Supabase Storage first; on any error it falls back to ./uploads
// disk (same naming + URL format as the legacy handlers) so uploads never
// break during transition. Failures are logged server-side, never leaked.
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
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	endpoint := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, bucket, filename)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(body, &payload)
		msg := payload.Message
		if msg == "" {
			msg = payload.Error
		}
		if msg == "" {
			msg = string(body)
		}
		return "", fmt.Errorf("storage upload (status %d): %s", resp.StatusCode, msg)
	}
	return s.publicURLFor(bucket, filename), nil
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
