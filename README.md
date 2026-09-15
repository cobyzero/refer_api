# Refer API — Go + Supabase Self-Hosted

Backend de **Refer App** (Flutter sin cambios): `Flutter → Go API → Supabase Auth + Supabase PostgreSQL`.
Go es el único que habla con Supabase; Flutter usa el mismo contrato `/api/*` de siempre.

## Arquitectura

```text
Flutter ──HTTPS──▶ Go :3000 ─┬──▶ Kong :8000 ─┬──▶ Auth :9999 ──┐
                             │                └──▶ REST :3000 ──┤
                             └──────────────────────────────────┴──▶ PostgreSQL :5432
```

| Servicio (compose) | Imagen | Puerto host |
|---|---|---|
| `app` (Go/Fiber + Bun) | build local | `${HOST_PORT:-3001}` → 3000 |
| `supabase-db` | `supabase/postgres:15.8.1.060` | `${SUPABASE_DB_EXTERNAL_PORT:-5433}` → 5432 |
| `supabase-auth` | `supabase/auth:v2.196.0` | interno :9999 (vía Kong `/auth/v1/*`) |
| `supabase-rest` | `postgrest/postgrest:v12.2.0` | interno :3000 (vía Kong `/rest/v1/*`) |
| `supabase-kong` | `kong:2.8.1` (DB-less, `supabase/kong.yml`) | `${SUPABASE_KONG_HTTP_PORT:-8000}`, `:8443` |
| `supabase-storage` | `supabase/storage-api:v1.73.0` | interno :5000 (vía Kong `/storage/v1/*`), volumen `supabase_storage_data` |
| `supabase-meta` | `supabase/postgres-meta:v0.99.0` | interno :8080 (lo usa Studio) |
| `supabase-studio` (panel) | `supabase/studio:2026.09.04-sha-5a67366` | `${SUPABASE_STUDIO_PORT:-3002}` → 3000 |

## Panel (Studio)

Abre **http://localhost:3002** → verás proyecto `refer`:

- **Table Editor**: las 20 tablas migradas (`users`, `orders`, `products`, …) con sus counts.
- **Authentication → Users**: los usuarios (`auth.users`), creados con el mismo UUID que `public.users`.
- **SQL Editor**: para correr queries de verificación.

> El panel es solo administración/dev: Flutter nunca lo usa y Go no lo necesita.
> En prod, restringe el puerto 3002 por red o proxy reverso con auth.

## Requisitos

- Docker + Docker Compose
- Go ≥ 1.25 (solo desarrollo local)
- Client tools Postgres (`pg_dump`, `pg_restore`, `psql`) — solo para migrar: `brew install libpq`

## 1. Levantar todo (desde cero)

```bash
cp .env.example .env          # y completa los valores (ver tabla abajo)
docker compose up -d --wait   # PG + Auth + REST + Kong + Go, esperando health
docker compose ps             # todo debe estar healthy/running
```

Comprobaciones:

```bash
curl http://localhost:3001/health              # Go      → {"status":"ok"}
curl http://localhost:8000/auth/v1/health      # Auth    → {"name":"GoTrue",...}
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8000/rest/v1/  # → 200
```

> Primera vez: `supabase/init/01-role-passwords.sh` asigna password al rol
> `authenticator` (PostgREST) automáticamente. Solo corre con volumen vacío.
> Si el volumen ya existía sin ese password:
> `docker exec -e PGPASSWORD=<pw> refer_supabase_db psql -U supabase_admin -d postgres -c "ALTER USER authenticator WITH PASSWORD '<pw>';"`
> y reinicia `supabase-rest`.

## 2. Variables de entorno

`.env` (gitignored, nunca commitear). Copia de `.env.example`:

| Var | Para qué | Ejemplo local |
|---|---|---|
| `PORT` / `HOST_PORT` | puerto Go | `3000` / `3001` |
| `DB_HOST`…`DB_NAME` | PG local Supabase (todo en docker): `localhost:5433/postgres` | legacy solo vía `LEGACY_DATABASE_URL` (migrador) |
| `DATABASE_URL` | **DB principal**: Supabase PG (tiene prioridad sobre `DB_*`) | `postgres://postgres:<pw>@localhost:5433/postgres?sslmode=disable` (local) · `...@supabase-db:5432/...` (en compose) |
| `APP_DATABASE_URL` | override de `DATABASE_URL` **dentro del compose** (red interna `supabase-db`); vacío = default interno. En Dokploy, setear explícito para cutover/legacy | solo compose |
| `APP_SUPABASE_URL` | override de `SUPABASE_URL` dentro del compose (`http://supabase-kong:8000`); `.env` trae `localhost` para `go run` local | solo compose |
| `LEGACY_DATABASE_URL` | origen del migrador (vacío = usa `DB_*`) | URL de la DB legacy |
| `JWT_SECRET` | valida sesiones legacy (transición) | no cambiar sin desloguear a todos |
| `SUPABASE_URL` | Auth para Go: en compose `http://supabase-kong:8000`, local `http://localhost:8000` | según contexto |
| `SUPABASE_ANON_KEY` / `SUPABASE_SERVICE_ROLE_KEY` | anon (Go) / admin (solo servidor, jamás Flutter/Git) | keys HS256 firmadas con el JWT secret |
| `SUPABASE_JWT_SECRET` | firma/validación JWT Supabase (≥32 chars) | rotar en prod + re-firmar keys |
| `SUPABASE_POSTGRES_PASSWORD` | password `postgres`/`authenticator` (solo letras/números) | distinto en prod |
| `SUPABASE_PUBLIC_URL` / `SUPABASE_SITE_URL` | links email Auth / redirects | `http://localhost:8000` / `http://localhost:3000` |
| `GOTRUE_JWT_EXP` / `SUPABASE_MAILER_AUTOCONFIRM` | expiry JWT / `false` exige OTP de verificación | `3600` / `false` |
| `SUPABASE_MAILER_OTP_EXP` | expiración del código OTP en segundos | `600` (10 min) |
| SMTP_* (Resend) | emails reales de Auth (verify/recovery): `SMTP_HOST=smtp.resend.com`, `SMTP_USER=resend`, `SMTP_PASS=re_...` | ver `.env.example` |

Rotar en prod: `openssl rand -hex 16` (password), `openssl rand -base64 48` (JWT secret),
re-firmar ANON/SERVICE_ROLE HS256 (`iss: supabase-demo`, `exp` largo).

## 3. Migración Fase 2 (legacy → Supabase) — orden estricto

> Nunca destructiva: legacy jamás se escribe ni se borra. Cada paso es re-ejecutable.
> Usa `sh scripts/run-with-env.sh <cmd>` (carga `.env` tolerando caracteres especiales).

```bash
sh scripts/run-with-env.sh go run ./cmd/supabase-migrate -step=backup   # pg_dump -Fc → ./backups (solo lectura)
sh scripts/run-with-env.sh go run ./cmd/supabase-migrate -step=migrate  # dry-run: plan + counts, no escribe
sh scripts/run-with-env.sh go run ./cmd/supabase-migrate -step=migrate -apply  # schema+datos, auth.users, remap ids
sh scripts/run-with-env.sh go run ./cmd/supabase-migrate -step=validate # counts + FK huérfanas + auth parity → PASSED
make supabase-rls   # RLS deny-all, SOLO tras PASSED (requiere DATABASE_URL)
```

Notas:

- Passwords **no migran** (bcrypt vs GoTrue): se crea password aleatoria por usuario;
  con `-apply -send-recovery` se les envía email para definir la suya.
- Los `id` UUID se preservan (`auth.users.id == public.users.id`); si Auth ignora el id,
  se remapea en transacción con cascada (`orders`, `star_transactions`,
  `redeemed_rewards`, `cart_items`, `users_roles`).
- `validate` falla (exit ≠ 0) ante cualquier mismatch → **no hacer cutover**.
- Cutover prod: setear `DATABASE_URL`=<Supabase PG> en el entorno de `app`, probar la API
  (Flutter sin cambios), y solo entonces retirar la DB vieja (con backup previo).

Atajos: `make supabase-up`, `make migrate-backup|plan|apply|validate`, `make supabase-psql`.

## 4. Uploads (Supabase Storage)

`POST /api/products/upload` y `POST /api/users/upload` reciben multipart `file`
y responden `201 {imageUrl}` / `{photoUrl}` como siempre — Go guarda en Storage
(buckets `products`/`avatars`, lectura pública) con fallback transparente a disco
`./uploads` si Storage no está configurado. URLs nuevas:
`<public-url>/storage/v1/object/public/<bucket>/<file>`.

```bash
make supabase-storage-init  # buckets + policy (una vez por entorno)
make migrate-uploads        # sube ./uploads/* y reescribe URLs en DB (verificado: 0 refs viejas)
```

## 5. Auditoría (módulo admin de refer_web)

Toda mutación HTTP (`POST/PUT/PATCH/DELETE` en `/api/*`) queda registrada en
`audit_logs`: actor (JWT), acción, entidad, status, IP y detalle. Sockets del
carrito (`add/remove/quantity/clear`) también. Las lecturas no se loguean.

Acciones ricas: `auth.register|login(.failed)|forgot|reset|password`,
`order.create|status`, `user.create|update|delete|profile|photo`,
`product.create|update|photo`, `reward.redeem|create|update|delete`,
`collection.*`, `location.*`, `config.update`, `payment.intent`, `cart.*`.
El resto usa `METHOD /ruta` genérico. Nunca se loguean passwords ni secrets.

Solo ADMIN: `GET /api/audit-logs/?userId&action&q&entityType&entityId&from&to&page&limit`
→ `{data, total, page, limit}`; `GET /api/audit-logs/actions` → filtros del dropdown.

## 6. Endpoints (contrato Flutter, intacto)

`POST /api/auth/register|login|verify-email|resend-verification|forgot-password|reset-password|refresh|logout|change-password`,
`GET /api/auth/validate|me`, más `/users`, `/products`, `/collections`, `/stars` (+`GET /perks`),
`/dashboard`, `/search`, `/config`, `/locations`, `/payments`, `/orders`. Ver `routes/`.

### Verificación OTP por correo (Supabase Auth nativo + Resend)

1. `POST /api/auth/register` → `201 {user, needsVerification:true}` (sin tokens) + email con código de 6 dígitos (expira en `SUPABASE_MAILER_OTP_EXP`, default 10 min).
2. `POST /api/auth/verify-email {email, code}` → `200 {user, accessToken, refreshToken}`.
3. `POST /api/auth/resend-verification {email}` → reenvía el código (GoTrue aplica rate-limit).
4. `POST /api/auth/login` con cuenta no verificada → `403 {code: EMAIL_NOT_VERIFIED}` (no 401, para que Flutter abra la pantalla OTP en vez de "contraseña incorrecta").

Requiere `SUPABASE_MAILER_AUTOCONFIRM=false` + SMTP de Resend (`SMTP_PASS=re_...`).
Sin SMTP válido, el registro falla (GoTrue no puede enviar el correo).
La plantilla del correo vive en `supabase/templates/confirmation.html` (usa `{{ .Token }}`;
Go la sirve el propio Go en `/templates` y GoTrue la descarga por HTTP — ver
`GOTRUE_MAILER_TEMPLATES_CONFIRMATION` en el compose).

## 5. Seguridad / RLS

- `SERVICE_ROLE_KEY` solo en servidor (Go). Jamás en Flutter, Git, `.env` versionado, respuestas ni logs.
- `supabase/rls.sql`: RLS habilitado en las 20 tablas, **sin policies permisivas** → PostgREST
  directo denegado por defecto. Toda autorización vive en Go (JWT dual + `RequireRole("ADMIN")`).
- JWT dual: Supabase primero (`SUPABASE_JWT_SECRET`), fallback legacy (`JWT_SECRET`) para
  sesiones viejas; roles enriquecidos desde `public.users` (los JWT Supabase no traen roles app).
