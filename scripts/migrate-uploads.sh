#!/bin/sh
# Migra ./uploads/* a Supabase Storage (bucket `uploads`, mismos filenames) y
# reescribe las URLs en DB de `/uploads/<f>` a
# `/storage/v1/object/public/uploads/<f>` (mismo host, Flutter transparente).
#
# Uso: sh scripts/run-with-env.sh sh scripts/migrate-uploads.sh
# Requiere: curl, psql. Buckets creados via API (idempotente). Re-ejecutable:
# los objetos se suben con upsert y el rewrite es estable (segunda pasada = 0 cambios).
set -e

API="$SUPABASE_URL/storage/v1"
SVC="$SUPABASE_SERVICE_ROLE_KEY"
: "${SUPABASE_URL:?SUPABASE_URL requerido}"
: "${SUPABASE_SERVICE_ROLE_KEY:?SERVICE_ROLE_KEY requerido}"
: "${DATABASE_URL:?DATABASE_URL requerido}"

echo "== 1/4 buckets (uploads/products/avatars, publicos) =="
for B in uploads products avatars; do
  CODE=$(curl -s -m 15 -o /dev/null -w "%{http_code}" -X POST "$API/bucket" \
    -H "apikey: $SVC" -H "Authorization: Bearer $SVC" -H 'Content-Type: application/json' \
    -d "{\"id\":\"$B\",\"name\":\"$B\",\"public\":true}")
  echo "bucket $B -> $CODE"
done

echo "== 2/4 subida de archivos =="
N=0
for F in uploads/*; do
  [ -f "$F" ] || continue
  case "$F" in *.gitkeep) continue;; esac
  NAME=$(basename "$F")
  case "$NAME" in
    *.png) CT="image/png";;
    *.webp) CT="image/webp";;
    *.gif) CT="image/gif";;
    *) CT="image/jpeg";;
  esac
  CODE=$(curl -s -m 60 -o /dev/null -w "%{http_code}" -X POST "$API/object/uploads/$NAME" \
    -H "apikey: $SVC" -H "Authorization: Bearer $SVC" -H "Content-Type: $CT" \
    -H "x-upsert: true" --data-binary "@$F")
  echo "$NAME -> $CODE"
  N=$((N+1))
done
echo "archivos subidos: $N"

echo "== 3/4 rewrite de URLs en DB =="
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<'SQL'
UPDATE products SET "imageUrl" = replace("imageUrl", '/uploads/', '/storage/v1/object/public/uploads/')
  WHERE "imageUrl" LIKE '%/uploads/%' AND "imageUrl" NOT LIKE '%/storage/v1/object/public/uploads/%';
UPDATE users SET "photoUrl" = replace("photoUrl", '/uploads/', '/storage/v1/object/public/uploads/')
  WHERE "photoUrl" LIKE '%/uploads/%' AND "photoUrl" NOT LIKE '%/storage/v1/object/public/uploads/%';
UPDATE product_categories SET "iconUrl" = replace("iconUrl", '/uploads/', '/storage/v1/object/public/uploads/')
  WHERE "iconUrl" LIKE '%/uploads/%' AND "iconUrl" NOT LIKE '%/storage/v1/object/public/uploads/%';
UPDATE product_categories SET "imageUrl" = replace("imageUrl", '/uploads/', '/storage/v1/object/public/uploads/')
  WHERE "imageUrl" LIKE '%/uploads/%' AND "imageUrl" NOT LIKE '%/storage/v1/object/public/uploads/%';
UPDATE product_categories SET "bannerUrl" = replace("bannerUrl", '/uploads/', '/storage/v1/object/public/uploads/')
  WHERE "bannerUrl" LIKE '%/uploads/%' AND "bannerUrl" NOT LIKE '%/storage/v1/object/public/uploads/%';
UPDATE rewards SET "imageUrl" = replace("imageUrl", '/uploads/', '/storage/v1/object/public/uploads/')
  WHERE "imageUrl" LIKE '%/uploads/%' AND "imageUrl" NOT LIKE '%/storage/v1/object/public/uploads/%';
SQL

echo "== 4/4 validacion =="
psql "$DATABASE_URL" -t -c "
SELECT 'stale_refs', COUNT(*) FROM (
  SELECT \"imageUrl\" AS u FROM products WHERE \"imageUrl\" LIKE '%/uploads/%' AND \"imageUrl\" NOT LIKE '%/storage/v1/object/public/uploads/%'
  UNION ALL SELECT \"photoUrl\" FROM users WHERE \"photoUrl\" LIKE '%/uploads/%' AND \"photoUrl\" NOT LIKE '%/storage/v1/object/public/uploads/%'
  UNION ALL SELECT \"iconUrl\" FROM product_categories WHERE \"iconUrl\" LIKE '%/uploads/%' AND \"iconUrl\" NOT LIKE '%/storage/v1/object/public/uploads/%'
  UNION ALL SELECT \"imageUrl\" FROM product_categories WHERE \"imageUrl\" LIKE '%/uploads/%' AND \"imageUrl\" NOT LIKE '%/storage/v1/object/public/uploads/%'
  UNION ALL SELECT \"bannerUrl\" FROM product_categories WHERE \"bannerUrl\" LIKE '%/uploads/%' AND \"bannerUrl\" NOT LIKE '%/storage/v1/object/public/uploads/%'
  UNION ALL SELECT \"imageUrl\" FROM rewards WHERE \"imageUrl\" LIKE '%/uploads/%' AND \"imageUrl\" NOT LIKE '%/storage/v1/object/public/uploads/%'
) s;"
psql "$DATABASE_URL" -t -c "SELECT 'storage_objects_uploads', COUNT(*) FROM storage.objects WHERE bucket_id='uploads';"
echo "OK si stale_refs=0 y storage_objects_uploads == archivos subidos."
