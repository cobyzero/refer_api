package database

import (
	"context"
	"time"

	"refer_api/models"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

// EnsureAdmin creates the default admin user once (idempotent).
// Email: admin@admin.com  Password: admin123@
func EnsureAdmin(db *bun.DB) error {
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

	hashed, err := bcrypt.GenerateFromPassword([]byte("admin123@"), 10)
	if err != nil {
		return err
	}

	admin := &models.User{
		ID:          uuid.New().String(),
		Name:        "Admin",
		Email:       "admin@admin.com",
		Password:    string(hashed),
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
	return EnsureAdmin(db)
}
