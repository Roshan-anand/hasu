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

- [ ] wht is COLESCE in SQL
  - what is COALESCE in SQL
  - how it pairs with GROUP BY

- [ ] CSRF deep dive

- [ ] AES encryption
  - what is AES encryption
  - what is AES-256-GCM

- [ ] tanstack query lazy fetch for org switcher
  - how `enabled: false` + `refetch()` works for click-to-load dropdown data
  - when to update local store from query cache vs mutation response

- [ ] svelte feature-scoped class context
  - why combine feature state + UI state into one context class
  - when to use Symbol.for keys for context isolation

- [ ] SSE with EventSource for deployment logs
  - how to open and close `EventSource` safely with dialog lifecycle
  - how custom SSE event names (like `event`) map to `addEventListener` in browser

- [ ] feature-scoped query/mutation modules with page-local runes
  - move API contracts, payload types, and query keys into `src/lib/features/*`
  - keep `$state`/`$derived` form orchestration inside route components and pass reactive getters to query hooks
