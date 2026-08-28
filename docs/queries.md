# Queries

- [ ] what is embed.FS

  ```go
  //go:embed all:dist
  var embedded embed.FS

  var DistDirFS, _ = fs.Sub(embedded, "dist")
  ```

- [ ] pooling
  - what why when how pooling
  - simple example Go code for pooling

- [ ] how to production JWT
  - how to use JWT in production manner
  - what are best practices

- [ ] what is COLESCE in SQL
  - what is COALESCE in SQL
  - how it pairs with GROUP BY

- [ ] CSRF deep dive

- [ ] AES encryption
  - what is AES encryption
  - what is AES-256-GCM

- [ ] tanstack query lazy fetch for org switcher
  - how `enabled: false` + `refetch()` works for click-to-load dropdown data
  - when to update local store from query cache vs mutation response

- [ ] why we have a nextconfig package

- [x] Traefik YAML static config — mapping CLI flags to traefik.yml

  Source: [Traefik docs](https://doc.traefik.io/traefik/getting-started/configuration-overview/), [sample.yml](https://github.com/traefik/traefik/blob/master/traefik.sample.yml)

  **CLI → YAML mapping rules:**
  - `--entrypoints.web.address=:80` → `entryPoints: web: address: ":80"`
  - `--providers.swarm.endpoint=unix:///var/run/docker.sock` → `providers: swarm: endpoint: "unix:///var/run/docker.sock"`
  - `--api.insecure=true` / `--api.dashboard=true` → `api: insecure: true / dashboard: true`
  - `--certificatesresolvers.le.acme.email=x@y.com` → `certificatesResolvers: le: acme: email: "x@y.com"`
  - `--log.level=DEBUG` → `log: level: DEBUG`
  - `--accesslog=true` → `accessLog: {}`

  **Auto-discovery:** Traefik looks for `traefik.yml` in `/etc/traefik/`, `$XDG_CONFIG_HOME/`, `$HOME/.config/`, `.` (working dir). Override with `--configFile`.

  **Best practices:**
  - `exposedByDefault: false` + require `traefik.enable=true` labels
  - Mount docker socket read-only (`:ro`)
  - `api.insecure: true` only for dev
  - Persist `acme.json` on a volume
  - Test LE with staging CA first
  - YAML is static config only; dynamic config (routers, services, middlewares) goes in Docker labels or a separate file provider
