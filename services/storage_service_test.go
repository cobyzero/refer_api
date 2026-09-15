package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStorageService_DisabledWithoutEnv(t *testing.T) {
	os.Unsetenv("SUPABASE_URL")
	os.Unsetenv("SUPABASE_SERVICE_ROLE_KEY")
	svc := NewStorageService()
	if svc.Enabled() {
		t.Error("expected disabled without SUPABASE_URL/SERVICE_ROLE_KEY")
	}
}

func TestStorageService_SaveFallsBackToDisk(t *testing.T) {
	os.Unsetenv("SUPABASE_URL")
	os.Unsetenv("SUPABASE_SERVICE_ROLE_KEY")

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

func TestStorageService_PublicURLShape(t *testing.T) {
	os.Setenv("SUPABASE_URL", "http://supabase-kong:8000")
	os.Setenv("SUPABASE_PUBLIC_URL", "http://localhost:8000")
	os.Setenv("SUPABASE_SERVICE_ROLE_KEY", "svc")
	defer os.Unsetenv("SUPABASE_URL")
	defer os.Unsetenv("SUPABASE_PUBLIC_URL")
	defer os.Unsetenv("SUPABASE_SERVICE_ROLE_KEY")

	svc := NewStorageService().(*storageService)
	got := svc.publicURLFor(BucketAvatars, "f.jpg")
	want := "http://localhost:8000/storage/v1/object/public/avatars/f.jpg"
	if !strings.EqualFold(got, want) {
		t.Errorf("expected %s, got %s", want, got)
	}
}
