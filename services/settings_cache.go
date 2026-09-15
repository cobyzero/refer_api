package services

import (
	"sync"
	"time"
)

// Fase 0: tiny in-memory TTL cache for app_settings.
// The table has ~10 rows and is read on every GET /config plus on every
// auth register/login/forgot/resend (via authSettings). Caching 30s removes
// that hot read without any infra. Writes invalidate immediately.

var (
	settingsCacheMu  sync.RWMutex
	settingsCache    map[string]string
	settingsCacheExp time.Time
	settingsTTL      = 30 * time.Second
)

func getCachedSettings() (map[string]string, bool) {
	settingsCacheMu.RLock()
	defer settingsCacheMu.RUnlock()
	if settingsCache == nil || time.Now().After(settingsCacheExp) {
		return nil, false
	}
	out := make(map[string]string, len(settingsCache))
	for k, v := range settingsCache {
		out[k] = v
	}
	return out, true
}

func setCachedSettings(s map[string]string) {
	settingsCacheMu.Lock()
	defer settingsCacheMu.Unlock()
	out := make(map[string]string, len(s))
	for k, v := range s {
		out[k] = v
	}
	settingsCache = out
	settingsCacheExp = time.Now().Add(settingsTTL)
}

// InvalidateSettingsCache drops the cached settings (call after any Upsert).
func InvalidateSettingsCache() {
	settingsCacheMu.Lock()
	defer settingsCacheMu.Unlock()
	settingsCache = nil
	settingsCacheExp = time.Time{}
}
