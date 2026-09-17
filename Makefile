.PHONY: dev dev-docker build run watch help install-air up psql

AIR := $(HOME)/go/bin/air

help:
	@echo "Targets:"
	@echo "  make dev          - hot reload local con air (compila con CUALQUIER cambio .go/.html/.yaml)"
	@echo "  make dev-docker   - docker compose con watch (reconstruye imagen con cualquier cambio)"
	@echo "  make install-air  - instala air si no está"
	@echo "  make build        - compila binario ./tmp/main"
	@echo "  make run          - compila y ejecuta una vez"
	@echo "  Stack (Kratos + PG externa + R2):"
	@echo "  make up             - levanta Kratos (migra) + Go (espera healthy)"
	@echo "  make psql           - psql contra la PG externa (requiere DATABASE_URL)"

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

# --- Stack: Kratos + PG externa ---

up:
	docker compose up -d --wait kratos app
	docker compose ps

psql:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL requerido" && exit 1)
	psql "$(DATABASE_URL)"
