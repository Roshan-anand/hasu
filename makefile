.PHONY: env install check format build generate setup start stop cloud-tunnel test-backend test traefik-logs services-rm clean clean-web clean-server

env:
	@cd apps/server && cp .env.example .env && \
	cd apps/web && cp .env.example .env

install:
	@bun i && \
	turbo install

check:
	turbo check

format:
	turbo format

build:
	turbo build

generate:
	cd apps/server && \
	sqlc generate

setup: install build
	@turbo setup

# lcoal development cmds
start:
	turbo dev --ui tui --filter=web --filter=server

start-landing:
	turbo dev --ui tui --filter=landing --filter=@hasu/ui

start-all:
	turbo dev --ui tui

stop:
	@cd apps/server && \
	go run cmd/setup/main.go dev-stop && \
	pkill -f "turbo dev"

cloud-tunnel:
	cloudflared tunnel run hasu-server

test-backend:
	@cd apps/server && \
	go run cmd/setup/main.go test-backend

test:
	cd apps/server && \
	go test -race -v ./...

traefik-logs:
	@cd apps/server && \
	go run cmd/setup/main.go traefik-logs

services-rm:
	docker service rm hasu_traefik

# production server cmds
start-prod-build:
	docker compose -f docker/compose.prod.yml up --build

start-prod:
	docker compose -f docker/compose.prod.yml up

stop-prod:
	docker compose -f docker/compose.prod.yml down

# cleanup func to remove all node_modules and build artifacts
clean-web:
	@cd apps/web && \
	rm -rf node_modules .svelte-kit build

clean-server:
	@cd apps/server && \
	rm -rf bin frontend/dist bin data

clean: clean-web clean-server
	@rm -rf node_modules
