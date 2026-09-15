package services

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// DefaultAppVersion is the fallback when no version is configured.
const DefaultAppVersion = "1.0.0"

// Setting keys for the store-facing versions (app_settings table).
const (
	SettingAppVersion    = "app_version"
	SettingAppMinVersion = "app_min_version"
)

// versionPattern accepts "1", "1.0" and "1.0.0" (numeric parts only).
var versionPattern = regexp.MustCompile(`^\d+(\.\d+){0,2}$`)

// IsValidVersion reports whether v is a numeric version string.
func IsValidVersion(v string) bool {
	return versionPattern.MatchString(strings.TrimSpace(v))
}

// ResolveAppVersions reads the seed versions from the environment
// (APP_VERSION / APP_MIN_VERSION). Used as defaults when the DB has no
// operator-configured values yet. APP_MIN_VERSION defaults to APP_VERSION.
func ResolveAppVersions() (version, minVersion string) {
	version = sanitizeVersion(os.Getenv("APP_VERSION"), DefaultAppVersion)
	minVersion = sanitizeVersion(os.Getenv("APP_MIN_VERSION"), version)
	if CompareVersions(minVersion, version) > 0 {
		log.Printf("config: APP_MIN_VERSION (%s) > APP_VERSION (%s); clamping to APP_VERSION", minVersion, version)
		minVersion = version
	}
	return version, minVersion
}

// sanitizeVersion trims and validates a version string, returning fallback
// (and logging) when empty or malformed.
func sanitizeVersion(raw, fallback string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fallback
	}
	if !IsValidVersion(v) {
		log.Printf("config: invalid version %q; falling back to %q", raw, fallback)
		return fallback
	}
	return v
}

// CompareVersions compares two numeric version strings ("1.2.0").
// Missing parts count as 0. Returns -1, 0 or 1.
func CompareVersions(a, b string) int {
	pa := strings.Split(strings.TrimSpace(a), ".")
	pb := strings.Split(strings.TrimSpace(b), ".")
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var va, vb int
		if i < len(pa) {
			va, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			vb, _ = strconv.Atoi(pb[i])
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	}
	return 0
}
