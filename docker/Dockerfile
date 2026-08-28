FROM docker:29.6.2-cli-alpine3.24 AS docker-cli

# build web static files
FROM dhi.io/bun:1-debian-dev AS web

WORKDIR /web

ENV NODE_ENV=development
ENV STATIC_OUT_PATH=./dist

COPY apps/web/package.json ./
COPY apps/web/bun.lock ./

COPY apps/web ./

RUN bun install --frozen-lockfile

RUN bun run build

# build server binary
FROM  golang:1.26-alpine AS builder

ENV SERVER_PUBLIC_URL=https://dev.roshananand.cloud
ENV APP_ENV=development

WORKDIR /server

RUN apk add --no-cache build-base

ENV SERVER_PUBLIC_URL=https://dev.roshananand.cloud

COPY apps/server/go.mod ./
COPY apps/server/go.sum ./

RUN go mod download && go mod verify

COPY apps/server ./
COPY --from=web ./web/dist ./frontend/dist

RUN CGO_ENABLED=1 GOOS=linux CC=gcc go build \
    -trimpath \
    -ldflags='-s -w' \
    -o ./bin/hasu ./cmd/server/main.go

# final image
FROM alpine AS final

RUN apk add --no-cache git openssh-client

WORKDIR /app

COPY --from=docker-cli /usr/local/bin/docker /usr/local/bin/docker
COPY --from=docker-cli /usr/local/libexec/docker/cli-plugins /usr/local/libexec/docker/cli-plugins

COPY --from=builder /server/bin/hasu ./

EXPOSE 8080

ENTRYPOINT ["/app/hasu"]
