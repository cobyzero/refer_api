# Refer API — Go + Kratos + R2

Backend de **Refer App** (Flutter sin cambios): `Flutter → Go API → Kratos (auth) + PostgreSQL + R2 (archivos)`.
Go es el único que habla con Kratos/R2; Flutter usa el mismo contrato `/api/*` de siempre.

## Arquitectura

```text
Flutter ──HTTPS──▶ Go :3000 ─┬──▶ Kratos :4433/4434 (auth, OTP 6 dígitos)
                              ├──▶ PostgreSQL :5433 (negocio + refresh tokens)
                              └──▶ R2 (uploads, fallback ./uploads)
```

| Servicio (compose) | Imagen | Puerto host |
|---|---|---|
| `app` (Go/Fiber + Bun) | build local | `${HOST_PORT:-3001}` → 3000 |
| `kratos-migrate` (one-shot) | `oryd/kratos:v26.2.0` | — (migra y sale) |
| `kratos` | `oryd/kratos:v26.2.0` | `127.0.0.1:4433` → 4433, `127.0.0.1:4434` → 4434 |

La PostgreSQL vive en proyecto Dokploy separado (`refer-db`):
se usa como PostgreSQL normal (databases `postgres` + `kratos`).

## Requisitos

- Docker + Docker Compose
- Go ≥ 1.25 (solo desarrollo local)
- `psql` (opcional, solo para inspeccionar la DB): `brew install libpq`

## 1. Levantar todo

```bash
cp .env.example .env          # y completa los valores (ver tabla abajo)
docker compose up -d --wait   # Kratos (migra) + Go, esperando health
docker compose ps             # todo debe estar running
```

Go crea el schema automáticamente al arrancar (tablas con `IF NOT EXISTS`,
incluida `refresh_tokens`) y el seed inicial. La data empieza desde cero,
sin migraciones manuales. Kratos migra sus propias tablas vía `kratos-migrate`.
Kratos es requerido: sin `KRATOS_PUBLIC_URL` + `KRATOS_ADMIN_URL` +
`AUTH_JWT_SECRET` el server no arranca (`Waiting for Kratos...` → `Kratos ready`).

Comprobaciones:

```bash
curl http://localhost:3001/health              # Go      → {"status":"ok"}
curl http://localhost:4434/health/ready        # Kratos  → {"status":"ok"}
```

## 2. Variables de entorno

`.env` (gitignored, nunca commitear). Copia de `.env.example`:

| Var | Para qué | Ejemplo local |
|---|---|---|
| `PORT` / `HOST_PORT` | puerto Go | `3000` / `3001` |
| `DB_HOST`…`DB_NAME` | PG externa | `149.202.62.87:5433/postgres` |
| `DATABASE_URL` | **DB principal** (tiene prioridad sobre `DB_*`) | `postgres://postgres:<pw>@149.202.62.87:5433/postgres?sslmode=disable` |
| `DB_PASSWORD` | password `postgres` (mismo que `refer-db`, fijado una vez) | distinto en prod |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | seed del admin (se aprovisiona en Kratos al arrancar) | `admin@admin.com` / cambiar en prod |
| `KRATOS_PUBLIC_URL` / `KRATOS_ADMIN_URL` | Kratos para Go: en compose `http://kratos:4433/4434`, local `http://localhost:4433/4434` | según contexto |
| `AUTH_JWT_SECRET` | firma/validación JWT de sesiones API (≥32 chars) | rotar en prod |
| `AUTH_JWT_EXP_SEC` / `AUTH_REFRESH_TTL_DAYS` | expiry access JWT / refresh opacos | `3600` / `30` |
| `KRATOS_SECRETS_CIPHER` / `KRATOS_SECRETS_COOKIE` | secretos Kratos (`openssl rand -hex 16/32`) | distintos en prod |
| `KRATOS_SMTP_URI` | SMTP courier Kratos. Ej Resend: `smtp://resend:re_...@smtp.resend.com:587/` | ver `.env.example` |
| `R2_ENDPOINT` / `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` | Cloudflare R2 (S3-compatible) | dashboard Cloudflare |
| `R2_PUBLIC_UPLOADS_URL` / `R2_PUBLIC_PRODUCTS_URL` / `R2_PUBLIC_AVATARS_URL` | base pública por bucket | dominio público R2 |

Rotar en prod: `openssl rand -hex 16` (passwords), `openssl rand -base64 48` (JWT secret).

Atajos: `make up`, `make psql`.

## 3. Uploads (Cloudflare R2)

`POST /api/products/upload` y `POST /api/users/upload` reciben multipart `file`
y responden `201 {imageUrl}` / `{photoUrl}` como siempre — Go guarda en R2
(buckets `products`/`avatars`, lectura pública) con fallback transparente a disco
`./uploads` si R2 no está configurado.

```bash
# En Cloudflare: crear buckets uploads/products/avatars con acceso público,
# API token con lectura+escritura, y pegar endpoint/keys/URLs en el .env.
```

## 4. Auditoría (módulo admin de refer_web)

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

## 5. Endpoints (contrato Flutter, intacto)

`POST /api/auth/register|login|verify-email|resend-verification|forgot-password|reset-password|refresh|logout|change-password`,
`GET /api/auth/validate|me`, más `/users`, `/products`, `/collections`, `/stars` (+`GET /perks`),
`/dashboard`, `/search`, `/config`, `/locations`, `/payments`, `/orders`. Ver `routes/`.

### Verificación OTP por correo (Kratos + Resend)

1. `POST /api/auth/register` → `201 {user, needsVerification:true}` (sin tokens) + email con código de 6 dígitos.
2. `POST /api/auth/verify-email {email, code}` → `200 {user, accessToken, refreshToken}`.
3. `POST /api/auth/resend-verification {email}` → reenvía el código.
4. `POST /api/auth/login` con cuenta no verificada → `403 {code: EMAIL_NOT_VERIFIED}` (no 401, para que Flutter abra la pantalla OTP en vez de "contraseña incorrecta").

Requiere `KRATOS_SMTP_URI` con credenciales Resend válidas.
Sin SMTP válido, el registro falla con `503 EMAIL_SERVICE_UNAVAILABLE`.
Kratos ownea credenciales (argon2) y códigos OTP; Go minta JWTs API cortos
(`AUTH_JWT_SECRET`) + refresh tokens opacos rotados en cada uso.

## 6. Seguridad

- `AUTH_JWT_SECRET`, `KRATOS_SECRETS_*`, `R2_SECRET_ACCESS_KEY`, `SMTP_PASS`, `STRIPE_SECRET_KEY` solo en servidor. Jamás en Flutter, Git, `.env` versionado, respuestas ni logs.
- Toda autorización vive en Go (JWT API + `RequireRole("ADMIN")`); roles enriquecidos desde `public.users` (los JWT no traen roles app).
- Access tokens cortos (default 1h) + refresh opacos single-use con revocación en logout/cambio/reset.
