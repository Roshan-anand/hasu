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

- [ ] badger db how query all logs by prefix works

- [ ] tanstack query cache keys per parent resource
  - why service lists should key by project_id now that org -> project -> service
  - how to avoid cache collisions when moving from org-scoped to project-scoped lists

- [ ] echo path params in handler tests
  - how to inject route params with `SetParamNames` + `SetParamValues`
  - see `backend/integration_tests/utils.go`
