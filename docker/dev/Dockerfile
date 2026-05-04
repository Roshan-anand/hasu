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
CMD ["bun", "dev", "--host", "0.0.0.0", "--port", "3000"]

# Stage 2: Backend development with go.
FROM --platform=$BUILDPLATFORM golang:1.26-trixie AS backend-dev

ENV SERVER_ENV=development

# Install development tools in a single layer
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    tzdata && \
    rm -rf /var/lib/apt/lists/* && \
    go install github.com/air-verse/air@latest

WORKDIR /app/backend

# Copy backend module files for dependency caching.
COPY backend/go.mod backend/go.sum ./

RUN go mod download && go mod verify

COPY ./backend ./

RUN mkdir -p frontend/dist && if [ ! -f frontend/dist/index.html ]; then printf '<!doctype html><html><body>dev placeholder</body></html>' > frontend/dist/index.html; fi;

EXPOSE 8080

ENTRYPOINT ["air"]
