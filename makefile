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

start:
	turbo dev --ui tui

stop:
	@pkill -f "make start" && \
	cd apps/server && \
	go run cmd/setup/main.go dev-stop

cloud-tunnel:
	cloudflared tunnel run dev

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
	docker service rm godploy_traefik

# cleanup func to remove all node_modules and build artifacts
clean-web:
	@cd apps/web && \
	rm -rf node_modules .svelte-kit build

clean-server:
	@cd apps/server && \
	rm -rf bin frontend/dist bin data

clean: clean-web clean-server
	@rm -rf node_modules
