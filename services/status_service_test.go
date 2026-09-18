package services

import (
	"context"
	"os"
	"testing"
)

func TestCheckStatus_NoDB_DegradedOrOutage(t *testing.T) {
	// Sin DB configurada el reporte debe marcar database down sin pánico.
	// R2 tampoco está configurado en el entorno de test.
	for _, k := range []string{
		"R2_ENDPOINT", "R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY",
		"KRATOS_PUBLIC_URL", "KRATOS_ADMIN_URL",
	} {
		os.Unsetenv(k)
	}
	report := CheckStatus(context.Background())
	if len(report.Services) == 0 {
		t.Fatal("expected services in report")
	}
	byName := map[string]ServiceStatus{}
	for _, s := range report.Services {
		byName[s.Name] = s
	}
	db, ok := byName["database"]
	if !ok {
		t.Fatal("expected database entry")
	}
	if db.Status != StatusDown {
		t.Errorf("expected database down without DB, got %q", db.Status)
	}
	if report.Status != OverallOutage {
		t.Errorf("expected outage without DB, got %q", report.Status)
	}
	if r2 := byName["r2"]; r2.Status != StatusDisabled {
		t.Errorf("expected r2 disabled without creds, got %q", r2.Status)
	}
}

func TestCheckStatus_Cache(t *testing.T) {
	a := CheckStatus(context.Background())
	b := CheckStatus(context.Background())
	if a.CheckedAt.IsZero() || b.CheckedAt.IsZero() {
		t.Fatal("expected timestamps")
	}
	if !a.CheckedAt.Equal(b.CheckedAt) {
		t.Error("expected cached report within TTL")
	}
}
