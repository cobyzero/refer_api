package services

import (
	"os"
	"path/filepath"
	"testing"
)

func clearR2Env(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"R2_ENDPOINT", "R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY",
		"R2_PUBLIC_UPLOADS_URL", "R2_PUBLIC_PRODUCTS_URL", "R2_PUBLIC_AVATARS_URL",
	} {
		os.Unsetenv(k)
	}
}

func TestStorageService_DisabledWithoutEnv(t *testing.T) {
	clearR2Env(t)
	svc := NewStorageService()
	if svc.Enabled() {
		t.Error("expected disabled without R2_ENDPOINT/R2_ACCESS_KEY_ID/R2_SECRET_ACCESS_KEY")
	}
}

func TestStorageService_SaveFallsBackToDisk(t *testing.T) {
	clearR2Env(t)

	dir := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(prev)

	svc := NewStorageService()
	url, err := svc.Save(BucketProducts, "abc123.jpg", []byte("fake-image"), "image/jpeg", "http://localhost:3000")
	if err != nil {
		t.Fatalf("expected disk fallback to succeed, got: %v", err)
	}
	if url != "http://localhost:3000/uploads/abc123.jpg" {
		t.Errorf("expected legacy URL shape, got: %s", url)
	}
	if _, err := os.Stat(filepath.Join(dir, "uploads", "abc123.jpg")); err != nil {
		t.Errorf("expected file on disk: %v", err)
	}
}

func TestStorageService_EnabledWithCreds(t *testing.T) {
	clearR2Env(t)
	os.Setenv("R2_ENDPOINT", "https://abc123.r2.cloudflarestorage.com")
	os.Setenv("R2_ACCESS_KEY_ID", "key")
	os.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	defer clearR2Env(t)

	svc := NewStorageService()
	if !svc.Enabled() {
		t.Error("expected enabled with R2 creds")
	}
}

func TestStorageService_UploadWithoutBucketURLFallsBack(t *testing.T) {
	clearR2Env(t)
	os.Setenv("R2_ENDPOINT", "https://abc123.r2.cloudflarestorage.com")
	os.Setenv("R2_ACCESS_KEY_ID", "key")
	os.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	defer clearR2Env(t)

	dir := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(prev)

	// Creds present but no public URL for the bucket (and no network):
	// must fall back to disk, never fail.
	svc := NewStorageService()
	url, err := svc.Save(BucketAvatars, "f.jpg", []byte("x"), "image/jpeg", "http://localhost:3000")
	if err != nil {
		t.Fatalf("expected disk fallback, got: %v", err)
	}
	if url != "http://localhost:3000/uploads/f.jpg" {
		t.Errorf("expected disk URL, got: %s", url)
	}
}
