# TODOs

## staging

- [ ] setup openAPI
- [ ] Setup logger
- [x] setup health check route
- [x] setup tests
- [ ] integration test for main func
- [ ] setup CI/CD
- [ ] setup linting and formatting for backend
- [ ] setup precommit hooks
- [x] write documentation
- [ ] setup monorepo
- [x] setup hot reloading in development
- [x] setup dev env for separate frontend
- [x] setup containerized dev env
- [x] migrate UI from vue to svelte
- [ ] validate CORS & cookie setting for both dev and prod[!IMP]

## application

- [ ] setup rate limiting middleware [medium, backend]
- [ ] delete project btn sould show popup to confirm deletion. [easy, ui]
- [x] git provider page. [easy, ui]
- [ ] tests for pqsl service lifecycle
- [ ] tests for badgerDB operations
- [ ] rememberMe functionality for login

## enhancements

- [ ] use dynamic imports in client side
- create service form
  - [x] the git provider github shoudl be auto fetched if the selected service is app. [easy, query]
  - [x] select field for selecting git-provider-app after **git-provider-selection**.
  - [ ] select field for selecting branch after **repo-selection**.
  - [x] input field for build path after **branch-selection**.
  - [ ] select field for selecting watch path after **build-path-selection**.
  - [x] update create service api to accept build path, repo nd branch. [easy, api]
  - [x] if no github app connected then show msg and link to connect github app. [easy, ui]
  - [ ] Load ui for select option till it fetch data
  - [ ] the name input shoudl be dafault to selected repo name.
- [ ] at [CheckUserExistsInOrg](../../backend/internal/handlers/utils.go) func return org details instead of just bool.

## Potential bugs

- [x] not using enums for column user.role in [sql](../../backend/sqlite/migrations/0001_init_schema.up.sql).
- [ ] service data is stored in DB and deployed, but what if user remove the service from terminal. the data still exists.
- [ ] the github app is stored linked to org_ig, what if user fails on instllation then data still ramins. so retry fails because there is multiple github app store in singe org.
- [ ] delete github app only deletes app from the DB and not from the github.
- [ ] form errors pops up as [object object]
- [ ] create and list org not working
- [ ] in create_service_form, if the name input have '/' in the string then this cause bug while creating a file for that code in the name of the service_name. so need to prevent user from entering '/' in the name input field. [easy, ui]
