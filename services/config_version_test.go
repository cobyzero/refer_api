package services

import (
	"errors"
	"testing"

	"refer_api/models"
	"refer_api/repository"
)

// =====================
// Mock ConfigRepository
// =====================
type mockConfigRepo struct {
	rows      []models.AppSetting
	findErr   error
	upsertErr error
	upserted  map[string]string
}

func newMockConfigRepo(rows []models.AppSetting) *mockConfigRepo {
	return &mockConfigRepo{rows: rows, upserted: map[string]string{}}
}

func (m *mockConfigRepo) FindAll() ([]models.AppSetting, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.rows, nil
}

func (m *mockConfigRepo) Upsert(key, value string) (*models.AppSetting, error) {
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	m.upserted[key] = value
	// Keep subsequent reads coherent.
	found := false
	for i, r := range m.rows {
		if r.Key == key {
			m.rows[i].Value = value
			found = true
		}
	}
	if !found {
		m.rows = append(m.rows, models.AppSetting{Key: key, Value: value})
	}
	return &models.AppSetting{Key: key, Value: value}, nil
}

var _ repository.ConfigRepository = (*mockConfigRepo)(nil)

func strptr(s string) *string { return &s }

func TestResolveAppVersionsDefaults(t *testing.T) {
	t.Setenv("APP_VERSION", "")
	t.Setenv("APP_MIN_VERSION", "")
	version, minVersion := ResolveAppVersions()
	if version != DefaultAppVersion {
		t.Fatalf("expected version %q, got %q", DefaultAppVersion, version)
	}
	if minVersion != DefaultAppVersion {
		t.Fatalf("expected minVersion %q, got %q", DefaultAppVersion, minVersion)
	}
}

func TestResolveAppVersionsFromEnv(t *testing.T) {
	t.Setenv("APP_VERSION", "2.3.0")
	t.Setenv("APP_MIN_VERSION", "2.0.0")
	version, minVersion := ResolveAppVersions()
	if version != "2.3.0" || minVersion != "2.0.0" {
		t.Fatalf("got version=%q minVersion=%q", version, minVersion)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0", "1.9.9", 1},
		{"1.0", "1.0.0", 0},
		{"1.10.0", "1.9.0", 1},
	}
	for _, tc := range cases {
		if got := CompareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestGetSettingsVersionsFromDB(t *testing.T) {
	repo := newMockConfigRepo([]models.AppSetting{
		{Key: SettingAppVersion, Value: "3.1.0"},
		{Key: SettingAppMinVersion, Value: "3.0.0"},
	})
	svc := NewConfigService(repo)
	settings, err := svc.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings[SettingAppVersion] != "3.1.0" || settings[SettingAppMinVersion] != "3.0.0" {
		t.Fatalf("got %v", settings)
	}
}

func TestUpdateAppVersionsOk(t *testing.T) {
	repo := newMockConfigRepo(nil)
	svc := NewConfigService(repo)
	settings, err := svc.UpdateAppVersions(strptr("2.0.0"), strptr("1.5.0"))
	if err != nil {
		t.Fatal(err)
	}
	if settings[SettingAppVersion] != "2.0.0" || settings[SettingAppMinVersion] != "1.5.0" {
		t.Fatalf("got %v", settings)
	}
}

func TestUpdateAppVersionsPartial(t *testing.T) {
	repo := newMockConfigRepo([]models.AppSetting{
		{Key: SettingAppVersion, Value: "2.0.0"},
		{Key: SettingAppMinVersion, Value: "1.0.0"},
	})
	svc := NewConfigService(repo)
	settings, err := svc.UpdateAppVersions(nil, strptr("1.8.0"))
	if err != nil {
		t.Fatal(err)
	}
	if settings[SettingAppVersion] != "2.0.0" || settings[SettingAppMinVersion] != "1.8.0" {
		t.Fatalf("got %v", settings)
	}
	if _, ok := repo.upserted[SettingAppVersion]; ok {
		t.Fatal("version should not have been upserted on partial update")
	}
}

func TestUpdateAppVersionsInvalid(t *testing.T) {
	repo := newMockConfigRepo(nil)
	svc := NewConfigService(repo)
	if _, err := svc.UpdateAppVersions(strptr("abc"), nil); !errors.Is(err, ErrVersionInvalid) {
		t.Fatalf("expected ErrVersionInvalid, got %v", err)
	}
}

func TestUpdateAppVersionsMinTooHigh(t *testing.T) {
	repo := newMockConfigRepo([]models.AppSetting{
		{Key: SettingAppVersion, Value: "2.0.0"},
	})
	svc := NewConfigService(repo)
	if _, err := svc.UpdateAppVersions(nil, strptr("3.0.0")); !errors.Is(err, ErrMinVersionTooHigh) {
		t.Fatalf("expected ErrMinVersionTooHigh, got %v", err)
	}
}
