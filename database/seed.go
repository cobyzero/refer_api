package database

import (
	"context"
	"time"

	"refer_api/models"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// EnsureAdmin creates the default admin user once (idempotent).
// The admin signs in via Supabase Auth: authUserID must be the auth.users.id
// provisioned beforehand (e.g. via Supabase EnsureUser), so public.users
// mirrors it and FKs stay consistent. An empty authUserID keeps a random UUID
// (local-only, cannot sign in until provisioned).
// Default credentials: admin@admin.com / admin123@ (override via ADMIN_EMAIL /
// ADMIN_PASSWORD env when provisioning).
func EnsureAdmin(db *bun.DB, authUserID string) error {
	ctx := context.Background()

	// 1. Ensure ADMIN role exists
	adminRole := &models.Role{}
	err := db.NewSelect().Model(adminRole).Where("name = ?", "ADMIN").Scan(ctx)
	if err != nil {
		// Not found -> create
		adminRole = &models.Role{
			ID:   uuid.New().String(),
			Name: "ADMIN",
		}
		if _, err := db.NewInsert().Model(adminRole).Exec(ctx); err != nil {
			return err
		}
	}

	// Also ensure USER role exists (needed for normal users, but keep minimal)
	userRole := &models.Role{}
	if err := db.NewSelect().Model(userRole).Where("name = ?", "USER").Scan(ctx); err != nil {
		userRole = &models.Role{
			ID:   uuid.New().String(),
			Name: "USER",
		}
		if _, err := db.NewInsert().Model(userRole).Exec(ctx); err != nil {
			return err
		}
	}

	// 2. Check if admin already exists
	exists, err := db.NewSelect().Model((*models.User)(nil)).Where("email = ?", "admin@admin.com").Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Passwords live in Supabase Auth only: no local hash is stored.
	adminID := authUserID
	if adminID == "" {
		adminID = uuid.New().String()
	}

	admin := &models.User{
		ID:          adminID,
		Name:        "Admin",
		Email:       "admin@admin.com",
		Password:    "",
		KeepUpdated: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if _, err := db.NewInsert().Model(admin).Exec(ctx); err != nil {
		return err
	}

	pivot := &models.UserRole{
		UserID: admin.ID,
		RoleID: adminRole.ID,
	}
	if _, err := db.NewInsert().Model(pivot).Exec(ctx); err != nil {
		return err
	}

	return nil
}

// Seed is kept for backwards compatibility but now just ensures admin (no mock data, no wipe).
func Seed(db *bun.DB) error {
	return EnsureAdmin(db, "")
}
