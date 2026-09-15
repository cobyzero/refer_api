.PHONY: dev dev-docker build run watch help install-air supabase-up supabase-psql supabase-rls supabase-storage-init

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
	@echo "  make supabase-rls      - aplica RLS restrictivo (tras el primer arranque)"
	@echo "  make supabase-storage-init - crea buckets + policy lectura pública"

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
