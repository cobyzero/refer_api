package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"refer_api/auth"
	"refer_api/database"
)

// Estados de cada dependencia en la página de estado.
const (
	StatusUp       = "up"
	StatusDown     = "down"
	StatusDisabled = "disabled"
)

// Estado global agregado.
const (
	OverallOperational = "operational"
	OverallDegraded    = "degraded"
	OverallOutage      = "outage"
)

var (
	appStartedAt = time.Now()
	statusHTTP   = &http.Client{Timeout: 8 * time.Second}
)

// ServiceStatus es el estado de una dependencia.
type ServiceStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	Detail    string `json:"detail,omitempty"`
}

// StatusReport es el agregado que exponen /status.json y la vista /status.
type StatusReport struct {
	Status    string          `json:"status"`
	CheckedAt time.Time       `json:"checked_at"`
	UptimeSec int64           `json:"uptime_sec"`
	Version   string          `json:"version"`
	Services  []ServiceStatus `json:"services"`
}

var (
	statusMu     sync.Mutex
	statusCached *StatusReport
	statusExp    time.Time
	statusTTL    = 10 * time.Second
)

// CheckStatus ejecuta los checks (con cache de 10s) y agrega el estado.
func CheckStatus(ctx context.Context) StatusReport {
	statusMu.Lock()
	if statusCached != nil && time.Now().Before(statusExp) {
		out := *statusCached
		statusMu.Unlock()
		return out
	}
	statusMu.Unlock()

	services := []ServiceStatus{
		{Name: "api", Status: StatusUp},
		checkDatabase(ctx),
		checkKratos(ctx, "kratos_public", auth.LoadConfig().PublicURL),
		checkKratos(ctx, "kratos_admin", auth.LoadConfig().AdminURL),
		checkR2(ctx),
	}

	overall := OverallOperational
	for _, s := range services {
		if s.Name == "database" && s.Status == StatusDown {
			overall = OverallOutage
			break
		}
		if s.Status == StatusDown {
			overall = OverallDegraded
		}
	}

	version := os.Getenv("APP_VERSION")
	if version == "" {
		version = "1.0.0"
	}
	report := StatusReport{
		Status:    overall,
		CheckedAt: time.Now().UTC(),
		UptimeSec: int64(time.Since(appStartedAt).Seconds()),
		Version:   version,
		Services:  services,
	}

	statusMu.Lock()
	statusCached = &report
	statusExp = time.Now().Add(statusTTL)
	statusMu.Unlock()
	return report
}

func checkDatabase(ctx context.Context) ServiceStatus {
	start := time.Now()
	if database.DB == nil {
		return ServiceStatus{Name: "database", Status: StatusDown, Detail: "sin conexión"}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := database.DB.PingContext(ctx); err != nil {
		return ServiceStatus{Name: "database", Status: StatusDown, Detail: "ping fallido"}
	}
	return ServiceStatus{Name: "database", Status: StatusUp, LatencyMs: time.Since(start).Milliseconds()}
}

func checkKratos(ctx context.Context, name, baseURL string) ServiceStatus {
	start := time.Now()
	if baseURL == "" {
		return ServiceStatus{Name: name, Status: StatusDisabled, Detail: "sin URL configurada"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health/ready", nil)
	if err != nil {
		return ServiceStatus{Name: name, Status: StatusDown, Detail: "request inválido"}
	}
	resp, err := statusHTTP.Do(req)
	if err != nil {
		return ServiceStatus{Name: name, Status: StatusDown, Detail: "no alcanzable"}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<12))
	if resp.StatusCode != 200 || !strings.Contains(string(body), "ok") {
		return ServiceStatus{Name: name, Status: StatusDown, Detail: fmt.Sprintf("http %d", resp.StatusCode)}
	}
	return ServiceStatus{Name: name, Status: StatusUp, LatencyMs: time.Since(start).Milliseconds()}
}

func checkR2(ctx context.Context) ServiceStatus {
	storage := NewStorageService()
	r2, ok := storage.(interface {
		Check(ctx context.Context) (int64, error)
	})
	if !ok || !storage.Enabled() {
		return ServiceStatus{Name: "r2", Status: StatusDisabled, Detail: "no configurado (fallback a disco)"}
	}
	if ms, err := r2.Check(ctx); err != nil {
		return ServiceStatus{Name: "r2", Status: StatusDown, Detail: "buckets inaccesibles"}
	} else {
		return ServiceStatus{Name: "r2", Status: StatusUp, LatencyMs: ms}
	}
}
