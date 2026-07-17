# TODOs

## staging

- [ ] setup openAPI
- [ ] Setup logger
- [x] setup health check route
- [x] setup tests
- [ ] integration test for main func
- [ ] setup CI/CD
- [x] setup linting and formatting for backend
- [ ] setup precommit hooks
- [x] write documentation
- [x] setup hot reloading in development
- [x] setup dev env for separate frontend
- [x] setup containerized dev env
- [x] migrate UI from vue to svelte
- [ ] validate CORS & cookie setting for both dev and prod[!IMP]

## application

- [ ] setup rate limiting middleware [medium, backend]
- [x] delete project btn sould show popup to confirm deletion. [easy, ui]
- [x] git provider page. [easy, ui]
- [x] tests for pqsl service lifecycle
- [ ] tests for badgerDB operations
- [x] test for application service lifecycle
- [ ] rememberMe functionality for login
- [x] make separate create page for app and DB service.
- [ ] use tiptap SDK for editor like ui for env management. [medium, ui]
- [x] update the project tests to include orphan volume cases
- [x] API & tests for orphan volume operations
- [x] switching org dosent refetch other query like get project, gh_app etc.
- [x] add watch_path docker file,context and build path settigns in app settings.
- [ ] Manual deletion of depdendency if service_id == target_id

## enhancements

- [ ] use dynamic imports in client side
- create service form
  - [x] the git provider github should be auto fetched if the selected service is app. [easy, query]
  - [x] select field for selecting git-provider-app after **git-provider-selection**.
  - [x] input field for build path after **branch-selection**.
  - [x] select field for selecting watch path after **build-path-selection**.
  - [x] update create service api to accept build path, repo nd branch. [easy, api]
  - [x] if no github app connected then show msg and link to connect github app. [easy, ui]
  - [x] Load ui for select option till it fetch data
  - [x] the name input should be dafault to selected repo name.
  - [ ] when setting name based on repo selected, also try to validate if service name already exists in client side itself to appen a random string for the name. [easy, ui]
- deployment logs page
  - [x] fix logs dialog box width
  - [x] show error msg in red bg.
  - [x] scroll should be always at the bottom. [easy, ui]
- rollback
  - [ ] config operation if image not exists
  - [ ] provide settings to +/- keeping the image of deployments
  - [ ] automaticcally delete deployments img if max exceeds
- [x] modify the bg workers to production level setup.
- [ ] when sidebar collapse the organization button avatar is oddly placed, fix by keeping it in center. [easy, ui]
- [x] if app service is internal then ask for port so in backend it automanically create internal url for internal communication between services.
- [ ] enhance log broker worker cunncurrency as per users
- [ ] enhance app service settings by controling few setting to check if service-exists in order to perform edits. 
- [x] gracefully handle redeploy when previous deployment is still in progress (also applies for new commit when prev dyp is still in progress)
- [ ] for every get<any>service API add a layer of verifying swarm_service exists. also for predef check vol also.
- [ ] add env is a text_area, make it a KV input fileds liek vercel
- [ ] auto remove image of old deploments.
- [ ] more deep module for predefined service creations (both in API and Preview worker)

## Potential bugs

- [ ] service data is stored in DB and deployed, but what if user remove the service from terminal. the data still exists.
- [ ] the github app is stored linked to org_ig, what if user fails on instllation then data still ramins. so retry fails because there is multiple github app store in singe org.
- [ ] delete github app only deletes app from the DB and not from the github.
- [x] form errors pops up as [object object]
- [x] create and list org not working
- [x] in create_service_form, if the name input have '/' in the string then this cause bug while creating a file for that code in the name of the service_name. so need to prevent user from entering '/' in the name input field. [easy, ui]
- [x] when view deployment logs after it is ended, or late subscribed then logs are not fully shown or have random logs.
- [ ] when deleting github app the every delete btn shows deleting if any one is clicked. [easy, ui]
- [x] if current dyp is in progress and either new commit trigger dyp or user click rebuild may leave a stale deployment worker.
