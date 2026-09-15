-- Supabase Storage init for Refer App.
--
-- Buckets (public read, write solo service_role via Go):
--   uploads   archivos legacy migrados tal cual (mismo filename)
--   products  fotos de productos (nuevos uploads)
--   avatars   fotos de perfil (nuevos uploads)
--
-- Go sube con SERVICE_ROLE (bypassa RLS). La policy permite lectura pública
-- directa (Flutter muestra <public-url>/storage/v1/object/public/<bucket>/<file>).
--
-- Aplicar:  make supabase-storage-init   (psql $DATABASE_URL -f supabase/storage.sql)
-- Re-ejecutable e idempotente.

INSERT INTO storage.buckets (id, name, public)
VALUES
  ('uploads',  'uploads',  true),
  ('products', 'products', true),
  ('avatars',  'avatars',  true)
ON CONFLICT (id) DO UPDATE SET public = true;

DROP POLICY IF EXISTS "public read" ON storage.objects;
CREATE POLICY "public read"
  ON storage.objects FOR SELECT
  USING (bucket_id IN ('uploads', 'products', 'avatars'));
