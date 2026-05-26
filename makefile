.PHONY: start reset restart build install-web install-server install check build-web build-bin generate test img-build setup dev services-rm web-logs server-logs traefik-logs cloud-tunnel clean clean-web clean-server clean-cache clean-all

permission:
	@sudo chown -R $(id -u):$(id -g) ./frontend/.svelte-kit

env:
	@cp .env.example .env && \
	cd frontend && cp .env.example .env
	
install-web:
	cd frontend && \
	bun install && \
	bun run prepare

install-server:
	cd backend && go mod tidy

install: install-web install-server

format-server:
	cd backend && go fmt ./...

format-web:
	cd frontend && bun run format && bun run lint

format: format-server format-web

check-web:
	cd frontend && bun run check

check-server:
	cd backend && go vet ./...

check: check-web check-server

build-web:
	cd frontend && \
	bun install && bun run build

build-bin:
	cd backend && \
	go mod tidy && \
	go build -o ../bin/godploy cmd/server/main.go

build: build-web build-bin

generate:
	cd backend && \
	sqlc generate

start: generate build
	@./bin/godploy

reset:
	rm -rf ./backend/data/* ./data/*

restart: reset start

test:
	cd backend && \
	go test -v ./...

setup: install build-web
	@cd backend && \
	go run cmd/setup/main.go setup

dev-start:
	@cd backend && \
	go run cmd/setup/main.go dev-start

server-start:
	@cd backend && \
	go run cmd/setup/main.go server-start

web-start:
	@cd frontend && \
	bun run dev --host 0.0.0.0 --port 3000

dev-stop:
	@cd backend && \
	go run cmd/setup/main.go dev-stop

services-rm:
	docker service rm godploy_traefik

test-backend:
	@cd backend && \
	go run cmd/setup/main.go test-backend

web-logs:
	@cd backend && \
	go run cmd/setup/main.go web-logs

server-logs:
	@cd backend && \
	go run cmd/setup/main.go server-logs

traefik-logs:
	@cd backend && \
	go run cmd/setup/main.go traefik-logs

cloud-tunnel:
	docker run --rm -it \
        --network host \
        cloudflare/cloudflared:latest \
        tunnel --no-autoupdate --url http://localhost:8080

clean-web:
	rm -rf ./frontend/node_modules ./frontend/.svelte-kit ./frontend/build

clean-server:
	rm -rf ./backend/bin ./backend/frontend/dist ./bin/godploy ./backend/data

clean: clean-web clean-server
