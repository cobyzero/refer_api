package handlers

import (
	"errors"
	"strconv"

	"refer_api/middleware"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// ConfigHandler coordinates HTTP requests for app configuration.
type ConfigHandler struct {
	configService services.ConfigService
}

// NewConfigHandler constructs a new ConfigHandler.
func NewConfigHandler(configService services.ConfigService) *ConfigHandler {
	return &ConfigHandler{configService: configService}
}

// GetConfig returns version, maintenance status, active features, system
// links, and the operator-controlled settings (e.g. currency).
// version/minVersion are operator-editable (web settings module) and stored
// in app_settings; APP_VERSION / APP_MIN_VERSION seed the defaults.
// The app forces an update below minVersion and suggests one below version.
func (h *ConfigHandler) GetConfig(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 30)
	settings, err := h.configService.GetSettings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{
		"version":     settings[services.SettingAppVersion],
		"minVersion":  settings[services.SettingAppMinVersion],
		"maintenance": false,
		"features": fiber.Map{
			"starsEnabled":         true,
			"referralsEnabled":     true,
			"authEnabled":          true,
			"smtpEnabled":          services.SmtpEnabled(settings),
			"signupEmailEnabled":   services.SignupEmailEnabled(settings),
			"recoveryEmailEnabled": services.RecoveryEmailEnabled(settings),
		},
		"endpoints": fiber.Map{
			"termsUrl":   "https://example.com/terms",
			"privacyUrl": "https://example.com/privacy",
			"supportUrl": "https://example.com/support",
		},
		"settings": settings,
	})
}

// UpdateConfig upserts operator-controlled settings. Accepts any subset of
// currency, taxEnabled, taxRate, serviceFeeRate, version, minVersion,
// smtpEnabled, signupEmailEnabled y recoveryEmailEnabled;
// only present fields are updated.
func (h *ConfigHandler) UpdateConfig(c *fiber.Ctx) error {
	var body struct {
		Currency             *string  `json:"currency"`
		TaxEnabled           *bool    `json:"taxEnabled"`
		TaxRate              *float64 `json:"taxRate"`
		ServiceFeeRate       *float64 `json:"serviceFeeRate"`
		Version              *string  `json:"version"`
		MinVersion           *string  `json:"minVersion"`
		SmtpEnabled          *bool    `json:"smtpEnabled"`
		SignupEmailEnabled   *bool    `json:"signupEmailEnabled"`
		RecoveryEmailEnabled *bool    `json:"recoveryEmailEnabled"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	settings, err := h.configService.GetSettings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	if body.Currency != nil {
		settings, err = h.configService.UpdateCurrency(*body.Currency)
		if err != nil {
			if errors.Is(err, services.ErrCurrencyInvalid) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"message":   err.Error(),
					"supported": services.SupportedCurrencies,
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	if body.TaxEnabled != nil || body.TaxRate != nil {
		enabled, taxRate := currentTax(settings)
		if body.TaxEnabled != nil {
			enabled = *body.TaxEnabled
		}
		if body.TaxRate != nil {
			taxRate = *body.TaxRate
		}
		settings, err = h.configService.UpdateTax(enabled, taxRate)
		if err != nil {
			if errors.Is(err, services.ErrTaxRateInvalid) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	if body.ServiceFeeRate != nil {
		settings, err = h.configService.UpdateServiceFee(*body.ServiceFeeRate)
		if err != nil {
			if errors.Is(err, services.ErrServiceFeeInvalid) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	if body.Version != nil || body.MinVersion != nil {
		settings, err = h.configService.UpdateAppVersions(body.Version, body.MinVersion)
		if err != nil {
			if errors.Is(err, services.ErrVersionInvalid) || errors.Is(err, services.ErrMinVersionTooHigh) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	if body.SmtpEnabled != nil || body.SignupEmailEnabled != nil || body.RecoveryEmailEnabled != nil {
		settings, err = h.configService.UpdateEmailSettings(body.SmtpEnabled, body.SignupEmailEnabled, body.RecoveryEmailEnabled)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	middleware.AttachAudit(c, "config.update", "config", "", map[string]interface{}{"currency": body.Currency, "taxEnabled": body.TaxEnabled, "taxRate": body.TaxRate, "serviceFeeRate": body.ServiceFeeRate, "version": body.Version, "minVersion": body.MinVersion, "smtpEnabled": body.SmtpEnabled, "signupEmailEnabled": body.SignupEmailEnabled, "recoveryEmailEnabled": body.RecoveryEmailEnabled})
	return c.JSON(fiber.Map{"settings": settings})
}

// currentTax reads the sanitized tax toggle/rate from a settings map.
func currentTax(settings map[string]string) (bool, float64) {
	enabled := services.DefaultTaxEnabled
	if v, err := strconv.ParseBool(settings["tax_enabled"]); err == nil {
		enabled = v
	}
	rate := services.DefaultTaxRate
	if v, err := strconv.ParseFloat(settings["tax_rate"], 64); err == nil && v >= 0 && v <= 1 {
		rate = v
	}
	return enabled, rate
}
