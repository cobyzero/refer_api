package services

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"refer_api/models"
	"refer_api/repository"
)

// Recovery codes preserve the Flutter forgot-password/reset-password
// {email, code} contract on top of Supabase Auth: the 6-digit code gates the
// reset, and the new password is then set in Supabase Auth (source of truth).
//
// Hardening: 6-digit codes are brute-forceable, so they get attempt limits,
// single-use semantics, uniform error messages (no oracle for "unknown email"
// vs "wrong code" vs "expired"), constant-time comparison, CSPRNG generation,
// and a resend cooldown.
const (
	resetCodeTTL         = 15 * time.Minute
	resetCodeCooldown    = 60 * time.Second
	resetCodeMaxAttempts = 5
	// minPasswordLength mirrors refer_app (passwordMinLength: 6).
	minPasswordLength = 6
)

// ErrRecoveryCodeInvalid is the ONLY error surfaced for missing, expired,
// exhausted or mismatched codes, so callers can't distinguish the cases.
var ErrRecoveryCodeInvalid = errors.New("invalid or expired recovery code")

type resetCodeInfo struct {
	code       string
	expiresAt  time.Time
	attempts   int
	lastSentAt time.Time
}

var resetCodes sync.Map // maps normalized email (string) to resetCodeInfo

// normalizeRecoveryEmail trims the lookup key so "a@x.com " can't hold a
// separate code slot from "a@x.com".
func normalizeRecoveryEmail(email string) string {
	return strings.TrimSpace(email)
}

// issueRecoveryCode mints a 6-digit code for a known user. Unknown emails get
// uniform success (anti-enumeration): same nil as for known users.
func issueRecoveryCode(userRepo repository.UserRepository, email string) error {
	key := normalizeRecoveryEmail(email)
	user, err := userRepo.FindByEmail(key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Anti-enumeration: same success as for known users.
			return nil
		}
		return err
	}
	if user == nil {
		return nil
	}

	// Resend cooldown: don't mint a fresh code (nor extend the window) when
	// one was just sent — cheapest spam/brute-force throttle.
	if val, ok := resetCodes.Load(key); ok {
		if info, ok := val.(resetCodeInfo); ok {
			if time.Since(info.lastSentAt) < resetCodeCooldown {
				return nil
			}
		}
	}

	// CSPRNG 6-digit code (math/rand is predictable across restarts).
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	now := time.Now()

	// Store code in memory with 15 minutes expiration
	resetCodes.Store(key, resetCodeInfo{
		code:       code,
		expiresAt:  now.Add(resetCodeTTL),
		lastSentAt: now,
	})

	// Recovery codes are credential-equivalent: never print them unless the
	// operator opts in explicitly (dev). Local devs set
	// LOG_RECOVERY_CODES=true to see the code.
	if os.Getenv("LOG_RECOVERY_CODES") == "true" {
		fmt.Printf("\n[RECOVERY CODE] Generated recovery code for %s: %s (expires in 15 mins)\n\n", user.Email, code)
	}
	return nil
}

// consumeRecoveryCode validates the code with constant-time comparison and
// burns it (single-use). Returns the user on success.
func consumeRecoveryCode(userRepo repository.UserRepository, email, code string) (*models.User, error) {
	key := normalizeRecoveryEmail(email)
	val, ok := resetCodes.Load(key)
	if !ok {
		return nil, ErrRecoveryCodeInvalid
	}

	info, ok := val.(resetCodeInfo)
	if !ok {
		resetCodes.Delete(key)
		return nil, ErrRecoveryCodeInvalid
	}
	if time.Now().After(info.expiresAt) || info.attempts >= resetCodeMaxAttempts {
		resetCodes.Delete(key)
		return nil, ErrRecoveryCodeInvalid
	}

	// Constant-time compare: no early-exit timing signal on prefix matches.
	if subtle.ConstantTimeCompare([]byte(info.code), []byte(code)) != 1 {
		info.attempts++
		if info.attempts >= resetCodeMaxAttempts {
			// Burn the code after too many guesses: the attacker must
			// trigger a new forgot-password (cooldown + audit trail).
			resetCodes.Delete(key)
		} else {
			resetCodes.Store(key, info)
		}
		return nil, ErrRecoveryCodeInvalid
	}

	user, err := userRepo.FindByEmail(key)
	if err != nil {
		return nil, ErrRecoveryCodeInvalid
	}
	if user == nil {
		resetCodes.Delete(key)
		return nil, ErrRecoveryCodeInvalid
	}

	// Single-use: clean up code from memory
	resetCodes.Delete(key)
	return user, nil
}
