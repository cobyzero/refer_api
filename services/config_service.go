package services

import (
	"errors"
	"strconv"
	"strings"

	"refer_api/repository"
)

// SupportedCurrencies is the allowlist of ISO 4217 codes the API accepts
// as display currency. Codes are stored upper-cased.
var SupportedCurrencies = []string{
	"USD", "MXN", "EUR", "COP", "PEN", "CLP", "ARS", "BRL", "GBP", "CAD",
}

// DefaultCurrency is used when no currency has been configured yet.
const DefaultCurrency = "USD"

// DefaultTaxRate is used when no tax rate has been configured yet (8%).
const DefaultTaxRate = 0.08

// DefaultTaxEnabled is used when the tax toggle has not been configured yet.
const DefaultTaxEnabled = true

// DefaultServiceFeeRate is used when no service fee has been configured yet (0%).
const DefaultServiceFeeRate = 0.0

// Email toggles (app_settings) for the settings module in refer_web.
// smtp_enabled is the master kill-switch for auth emails; the two granular
// flags control registro (signup verification) y olvido-contraseña (recovery)
// individually. Effective = master && granular. Defaults true (current behavior).
//
// SettingDirectConfirmEnabled gates the no-Stripe order confirmation path
// (testing only, toggled from the refer_web settings module). Default false:
// ConfirmOrder fails closed unless the operator explicitly enables it.
// NEVER true in prod — it settles orders without charging.
const (
	SettingSmtpEnabled           = "smtp_enabled"
	SettingSignupEmailEnabled    = "signup_email_enabled"
	SettingRecoveryEmailEnabled  = "recovery_email_enabled"
	DefaultSmtpEnabled           = true
	DefaultSignupEmailEnabled    = true
	DefaultRecoveryEmailEnabled  = true

	SettingDirectConfirmEnabled  = "direct_confirm_enabled"
	DefaultDirectConfirmEnabled  = false
)

// ErrCurrencyInvalid is returned when the currency code is not supported.
var ErrCurrencyInvalid = errors.New("Unsupported currency")

// ErrTaxRateInvalid is returned when the tax rate is outside 0..1.
var ErrTaxRateInvalid = errors.New("Tax rate must be between 0 and 1")

// ErrServiceFeeInvalid is returned when the service fee rate is outside 0..1.
var ErrServiceFeeInvalid = errors.New("Service fee rate must be between 0 and 1")

// ErrVersionInvalid is returned when a version string is not numeric (e.g. "1.0.0").
var ErrVersionInvalid = errors.New("Version must be numeric (e.g. 1.0.0)")

// ErrMinVersionTooHigh is returned when minVersion would exceed version.
var ErrMinVersionTooHigh = errors.New("Minimum version cannot be higher than latest version")

// ConfigService defines business logic for operator-controlled app settings.
type ConfigService interface {
	GetSettings() (map[string]string, error)
	UpdateCurrency(code string) (map[string]string, error)
	UpdateTax(enabled bool, rate float64) (map[string]string, error)
	UpdateServiceFee(rate float64) (map[string]string, error)
	// UpdateAppVersions upserts the store-facing versions. Either pointer may
	// be nil (partial update); the effective pair must satisfy min <= version.
	UpdateAppVersions(version, minVersion *string) (map[string]string, error)
	// UpdateEmailSettings upserts the SMTP/auth-email toggles. Any nil pointer
	// is left untouched (partial update desde el módulo de configuraciones).
	UpdateEmailSettings(smtpEnabled, signupEnabled, recoveryEnabled *bool) (map[string]string, error)
	// UpdateDirectConfirm upserts the testing-only direct-confirm toggle.
	UpdateDirectConfirm(enabled bool) (map[string]string, error)
}

type configService struct {
	configRepo repository.ConfigRepository
}

// NewConfigService constructs a new ConfigService.
func NewConfigService(configRepo repository.ConfigRepository) ConfigService {
	return &configService{configRepo: configRepo}
}

func (s *configService) GetSettings() (map[string]string, error) {
	// Fase 0: 30s in-memory cache for the ~10-row hot table.
	if cached, ok := getCachedSettings(); ok {
		return cached, nil
	}
	envVersion, envMinVersion := ResolveAppVersions()
	settings := map[string]string{
		"currency":                  DefaultCurrency,
		"tax_enabled":               strconv.FormatBool(DefaultTaxEnabled),
		"tax_rate":                  strconv.FormatFloat(DefaultTaxRate, 'f', -1, 64),
		"service_fee_rate":          strconv.FormatFloat(DefaultServiceFeeRate, 'f', -1, 64),
		SettingAppVersion:           envVersion,
		SettingAppMinVersion:        envMinVersion,
		SettingSmtpEnabled:          strconv.FormatBool(DefaultSmtpEnabled),
		SettingSignupEmailEnabled:   strconv.FormatBool(DefaultSignupEmailEnabled),
		SettingRecoveryEmailEnabled: strconv.FormatBool(DefaultRecoveryEmailEnabled),
		SettingDirectConfirmEnabled: strconv.FormatBool(DefaultDirectConfirmEnabled),
	}
	rows, err := s.configRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		settings[row.Key] = row.Value
	}
	if !isSupportedCurrency(settings["currency"]) {
		settings["currency"] = DefaultCurrency
	}
	if _, ok := parseFraction(settings["tax_rate"]); !ok {
		settings["tax_rate"] = strconv.FormatFloat(DefaultTaxRate, 'f', -1, 64)
	}
	if _, err := strconv.ParseBool(settings["tax_enabled"]); err != nil {
		settings["tax_enabled"] = strconv.FormatBool(DefaultTaxEnabled)
	}
	if _, ok := parseFraction(settings["service_fee_rate"]); !ok {
		settings["service_fee_rate"] = strconv.FormatFloat(DefaultServiceFeeRate, 'f', -1, 64)
	}
	settings[SettingSmtpEnabled] = sanitizeBool(settings[SettingSmtpEnabled], DefaultSmtpEnabled)
	settings[SettingSignupEmailEnabled] = sanitizeBool(settings[SettingSignupEmailEnabled], DefaultSignupEmailEnabled)
	settings[SettingRecoveryEmailEnabled] = sanitizeBool(settings[SettingRecoveryEmailEnabled], DefaultRecoveryEmailEnabled)
	settings[SettingDirectConfirmEnabled] = sanitizeBool(settings[SettingDirectConfirmEnabled], DefaultDirectConfirmEnabled)
	// Stored versions always win over env seeds; sanitize in case they were
	// written outside the validated path, keeping min <= version coherent.
	settings[SettingAppVersion] = sanitizeVersion(settings[SettingAppVersion], envVersion)
	settings[SettingAppMinVersion] = sanitizeVersion(settings[SettingAppMinVersion], settings[SettingAppVersion])
	if CompareVersions(settings[SettingAppMinVersion], settings[SettingAppVersion]) > 0 {
		settings[SettingAppMinVersion] = settings[SettingAppVersion]
	}
	setCachedSettings(settings)
	return settings, nil
}

func (s *configService) UpdateCurrency(code string) (map[string]string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if !isSupportedCurrency(code) {
		return nil, ErrCurrencyInvalid
	}
	if _, err := s.configRepo.Upsert("currency", code); err != nil {
		return nil, err
	}
	InvalidateSettingsCache()
	return s.GetSettings()
}

// UpdateTax upserts the tax toggle and rate (0..1 fraction, e.g. 0.08).
func (s *configService) UpdateTax(enabled bool, rate float64) (map[string]string, error) {
	if rate < 0 || rate > 1 {
		return nil, ErrTaxRateInvalid
	}
	if _, err := s.configRepo.Upsert("tax_enabled", strconv.FormatBool(enabled)); err != nil {
		return nil, err
	}
	if _, err := s.configRepo.Upsert("tax_rate", strconv.FormatFloat(rate, 'f', -1, 64)); err != nil {
		return nil, err
	}
	InvalidateSettingsCache()
	return s.GetSettings()
}

// UpdateServiceFee upserts the service fee rate (0..1 fraction, e.g. 0.05).
func (s *configService) UpdateServiceFee(rate float64) (map[string]string, error) {
	if rate < 0 || rate > 1 {
		return nil, ErrServiceFeeInvalid
	}
	if _, err := s.configRepo.Upsert("service_fee_rate", strconv.FormatFloat(rate, 'f', -1, 64)); err != nil {
		return nil, err
	}
	InvalidateSettingsCache()
	return s.GetSettings()
}

// UpdateAppVersions upserts the store-facing versions (e.g. "1.0.0").
// Either pointer may be nil for a partial update. The effective pair
// (provided values over current ones) must satisfy min <= version.
func (s *configService) UpdateAppVersions(version, minVersion *string) (map[string]string, error) {
	if version != nil && !IsValidVersion(*version) {
		return nil, ErrVersionInvalid
	}
	if minVersion != nil && !IsValidVersion(*minVersion) {
		return nil, ErrVersionInvalid
	}
	current, err := s.GetSettings()
	if err != nil {
		return nil, err
	}
	effectiveVersion := current[SettingAppVersion]
	if version != nil {
		effectiveVersion = strings.TrimSpace(*version)
	}
	effectiveMin := current[SettingAppMinVersion]
	if minVersion != nil {
		effectiveMin = strings.TrimSpace(*minVersion)
	}
	if CompareVersions(effectiveMin, effectiveVersion) > 0 {
		return nil, ErrMinVersionTooHigh
	}
	if version != nil {
		if _, err := s.configRepo.Upsert(SettingAppVersion, effectiveVersion); err != nil {
			return nil, err
		}
	}
	if minVersion != nil {
		if _, err := s.configRepo.Upsert(SettingAppMinVersion, effectiveMin); err != nil {
			return nil, err
		}
	}
	InvalidateSettingsCache()
	return s.GetSettings()
}

// UpdateEmailSettings upserts the SMTP/auth-email toggles (partial update).
func (s *configService) UpdateEmailSettings(smtpEnabled, signupEnabled, recoveryEnabled *bool) (map[string]string, error) {
	if smtpEnabled != nil {
		if _, err := s.configRepo.Upsert(SettingSmtpEnabled, strconv.FormatBool(*smtpEnabled)); err != nil {
			return nil, err
		}
	}
	if signupEnabled != nil {
		if _, err := s.configRepo.Upsert(SettingSignupEmailEnabled, strconv.FormatBool(*signupEnabled)); err != nil {
			return nil, err
		}
	}
	if recoveryEnabled != nil {
		if _, err := s.configRepo.Upsert(SettingRecoveryEmailEnabled, strconv.FormatBool(*recoveryEnabled)); err != nil {
			return nil, err
		}
	}
	InvalidateSettingsCache()
	return s.GetSettings()
}

// UpdateDirectConfirm upserts the testing-only direct-confirm toggle.
// Fail-closed default (false); the operator enables it from refer_web
// settings only for local/testing stacks. Never true in prod.
func (s *configService) UpdateDirectConfirm(enabled bool) (map[string]string, error) {
	if _, err := s.configRepo.Upsert(SettingDirectConfirmEnabled, strconv.FormatBool(enabled)); err != nil {
		return nil, err
	}
	InvalidateSettingsCache()
	return s.GetSettings()
}

// DirectConfirmEnabled reports whether the operator allowed no-Stripe order
// confirmation (testing only, default false). Fail-closed on any doubt.
func DirectConfirmEnabled(settings map[string]string) bool {
	return parseBoolSetting(settings, SettingDirectConfirmEnabled, DefaultDirectConfirmEnabled)
}

// sanitizeBool normalizes a stored bool string, falling back to def.
func sanitizeBool(raw string, def bool) string {
	if v, err := strconv.ParseBool(strings.TrimSpace(raw)); err == nil {
		return strconv.FormatBool(v)
	}
	return strconv.FormatBool(def)
}

// parseBoolSetting reads a bool from a sanitized settings map.
func parseBoolSetting(settings map[string]string, key string, def bool) bool {
	if settings == nil {
		return def
	}
	if v, err := strconv.ParseBool(strings.TrimSpace(settings[key])); err == nil {
		return v
	}
	return def
}

// SmtpEnabled reports the master kill-switch (default true).
func SmtpEnabled(settings map[string]string) bool {
	return parseBoolSetting(settings, SettingSmtpEnabled, DefaultSmtpEnabled)
}

// SignupEmailEnabled reports the effective flag for registro:
// master smtp_enabled && granular signup_email_enabled.
func SignupEmailEnabled(settings map[string]string) bool {
	return SmtpEnabled(settings) && parseBoolSetting(settings, SettingSignupEmailEnabled, DefaultSignupEmailEnabled)
}

// RecoveryEmailEnabled reports the effective flag for olvido-contraseña:
// master smtp_enabled && granular recovery_email_enabled.
func RecoveryEmailEnabled(settings map[string]string) bool {
	return SmtpEnabled(settings) && parseBoolSetting(settings, SettingRecoveryEmailEnabled, DefaultRecoveryEmailEnabled)
}

// parseFraction parses a 0..1 fraction stored as string.
func parseFraction(s string) (float64, bool) {
	rate, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || rate < 0 || rate > 1 {
		return 0, false
	}
	return rate, true
}

func isSupportedCurrency(code string) bool {
	for _, c := range SupportedCurrencies {
		if c == code {
			return true
		}
	}
	return false
}
