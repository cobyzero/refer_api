package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/uptrace/bun"
)

// EnsureStorage replica supabase/storage.sql de forma idempotente al arrancar:
// buckets uploads/products/avatars públicos + policy "public read".
// Usa la conexión del server (postgres, bypasa RLS como el service_role).
func EnsureStorage(db *bun.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, `
INSERT INTO storage.buckets (id, name, public)
VALUES
  ('uploads',  'uploads',  true),
  ('products', 'products', true),
  ('avatars',  'avatars',  true)
ON CONFLICT (id) DO UPDATE SET public = true;`); err != nil {
		return fmt.Errorf("ensure storage buckets: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DROP POLICY IF EXISTS "public read" ON storage.objects;`); err != nil {
		return fmt.Errorf("drop storage policy: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
CREATE POLICY "public read"
  ON storage.objects FOR SELECT
  USING (bucket_id IN ('uploads', 'products', 'avatars'));`); err != nil {
		return fmt.Errorf("create storage policy: %w", err)
	}
	log.Println("Storage buckets ensured (uploads/products/avatars, public read)")
	return nil
}
