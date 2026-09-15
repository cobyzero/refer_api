.PHONY: dev dev-docker build run watch help install-air migrate-backup migrate-plan migrate-apply migrate-validate supabase-up supabase-psql supabase-rls supabase-storage-init migrate-uploads

AIR := $(HOME)/go/bin/air

help:
	@echo "Targets:"
	@echo "  make dev          - hot reload local con air (compila con CUALQUIER cambio .go/.html/.yaml)"
	@echo "  make dev-docker   - docker compose con watch (reconstruye imagen con cualquier cambio)"
	@echo "  make install-air  - instala air si no está"
	@echo "  make build        - compila binario ./tmp/main"
	@echo "  make run          - compila y ejecuta una vez"
	@echo "  Supabase self-hosted:"
	@echo "  make supabase-up       - levanta PG + Auth + PostgREST + Kong (espera healthy)"
	@echo "  make supabase-psql     - psql contra el PG de Supabase (requiere DATABASE_URL)"
	@echo "  make supabase-rls      - aplica RLS restrictivo (SOLO tras validate OK)"
	@echo "  make supabase-storage-init - crea buckets + policy lectura pública"
	@echo "  make migrate-uploads   - sube ./uploads a Storage y reescribe URLs en DB"
	@echo "  Migración Fase 2 (orden: backup -> plan -> apply -> validate):"
	@echo "  make migrate-backup    - pg_dump -Fc de la DB legacy a ./backups (solo lectura)"
	@echo "  make migrate-plan      - dry-run: muestra counts y pasos sin escribir nada"
	@echo "  make migrate-apply     - restaura schema+datos, sincroniza auth.users, remapea ids"
	@echo "  make migrate-validate  - reporte parity: counts, FK huérfanas, auth vs public"

install-air:
	@if [ ! -x "$(AIR)" ]; then \
		echo ">> Instalando air-verse/air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	@$(AIR) -v 2>&1 | grep -v "^$$" | head -n2

dev: install-air
	@echo ">> Hot reload con air (watch: .go, .html, .tpl, .yaml, .json) ..."
	@echo ">> Cualquier cambio en handlers/, models/, routes/, etc. -> recompila automático"
	$(AIR)

dev-docker:
	@echo ">> Docker Compose Watch - rebuild automático con cualquier cambio ..."
	docker compose watch

dev-docker-up:
	docker compose up --watch

build:
	go build -o ./tmp/main .

run: build
	./tmp/main

# --- Supabase self-hosted ---

supabase-up:
	docker compose up -d --wait supabase-db supabase-auth supabase-rest supabase-kong
	docker compose ps

supabase-psql:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL requerido" && exit 1)
	psql "$(DATABASE_URL)"

supabase-rls:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL requerido" && exit 1)
	@echo "Aplicando RLS restrictivo (deny-all PostgREST directo)..."
	psql "$(DATABASE_URL)" -f supabase/rls.sql

supabase-storage-init:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL requerido" && exit 1)
	@echo "Creando buckets + policy de lectura pública..."
	psql "$(DATABASE_URL)" -f supabase/storage.sql

migrate-uploads:
	sh scripts/run-with-env.sh sh scripts/migrate-uploads.sh

# --- Migración Fase 2 (no destructiva: legacy nunca se escribe ni se borra) ---

migrate-backup:
	go run ./cmd/supabase-migrate -step=backup

migrate-plan:
	go run ./cmd/supabase-migrate -step=migrate

migrate-apply:
	go run ./cmd/supabase-migrate -step=migrate -apply

migrate-validate:
	go run ./cmd/supabase-migrate -step=validate
