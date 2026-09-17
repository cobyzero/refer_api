package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"refer_api/database"
	"refer_api/models"
)

// ErrRefreshInvalid is the only error surfaced for missing/expired refresh
// tokens (uniform, no oracle).
var ErrRefreshInvalid = errors.New("invalid or expired refresh token")

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IssueRefreshToken mints an opaque token and stores its sha256 hash.
func IssueRefreshToken(userID string, ttlDays int) (string, error) {
	if userID == "" {
		return "", errors.New("user id is required")
	}
	if ttlDays <= 0 {
		ttlDays = 30
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	now := time.Now()
	rt := &models.RefreshToken{
		TokenHash: hashToken(token),
		UserID:    userID,
		ExpiresAt: now.Add(time.Duration(ttlDays) * 24 * time.Hour),
		CreatedAt: now,
	}
	if _, err := database.DB.NewInsert().Model(rt).Exec(context.Background()); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}
	return token, nil
}

// ConsumeRefreshToken validates a token (constant-time), burns it
// (single-use rotation) and returns the user id.
func ConsumeRefreshToken(token string) (string, error) {
	if token == "" {
		return "", ErrRefreshInvalid
	}
	want := hashToken(token)
	var found []models.RefreshToken
	if err := database.DB.NewSelect().Model(&found).
		Where("token_hash = ?", want).Limit(1).
		Scan(context.Background()); err != nil {
		return "", ErrRefreshInvalid
	}
	if len(found) == 0 {
		return "", ErrRefreshInvalid
	}
	rt := found[0]
	if subtle.ConstantTimeCompare([]byte(rt.TokenHash), []byte(want)) != 1 {
		return "", ErrRefreshInvalid
	}
	if time.Now().After(rt.ExpiresAt) {
		_, _ = database.DB.NewDelete().Model((*models.RefreshToken)(nil)).
			Where("token_hash = ?", want).Exec(context.Background())
		return "", ErrRefreshInvalid
	}
	_, _ = database.DB.NewDelete().Model((*models.RefreshToken)(nil)).
		Where("token_hash = ?", want).Exec(context.Background())
	return rt.UserID, nil
}

// RevokeUserTokens deletes all refresh tokens of a user (logout, password
// change/reset). Never fails the caller: best-effort cleanup.
func RevokeUserTokens(userID string) {
	if userID == "" || database.DB == nil {
		return
	}
	_, _ = database.DB.NewDelete().Model((*models.RefreshToken)(nil)).
		Where("user_id = ?", userID).Exec(context.Background())
}
