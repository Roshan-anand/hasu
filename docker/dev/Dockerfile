# Stage 1: Frontend development with Bun.
FROM --platform=$BUILDPLATFORM oven/bun:1.2.23-slim AS frontend-dev

ENV NODE_ENV=development
ENV VITE_APP_ENV=development

WORKDIR /app/frontend
RUN chown -R bun:bun /app/frontend

USER bun

COPY --chown=bun:bun frontend/package.json frontend/bun.lock ./

RUN bun install --frozen-lockfile
RUN bun prepare

COPY --chown=bun:bun ./frontend ./

EXPOSE 3000
ENTRYPOINT ["bun", "dev", "--host", "0.0.0.0", "--port", "3000"]

FROM dhi.io/docker:29-cli-dev AS docker-cli

# Stage 2: Backend development with go.
FROM --platform=$BUILDPLATFORM golang:1.26-trixie AS backend-dev

ENV SERVER_ENV=development

# Install development tools and docker CLI + buildx for build jobs.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    tzdata && \
    rm -rf /var/lib/apt/lists/* && \
    go install github.com/air-verse/air@latest

COPY --from=docker-cli /usr/local/bin/docker /usr/local/bin/docker
COPY --from=docker-cli /usr/local/libexec/docker/cli-plugins /usr/local/libexec/docker/cli-plugins

WORKDIR /app/backend

# Copy backend module files for dependency caching.
COPY backend/go.mod backend/go.sum ./

RUN go mod download && go mod verify

COPY ./backend ./

RUN mkdir -p frontend/dist && if [ ! -f frontend/dist/index.html ]; then printf '<!doctype html><html><body>dev placeholder</body></html>' > frontend/dist/index.html; fi;

EXPOSE 8080

CMD ["air"]
