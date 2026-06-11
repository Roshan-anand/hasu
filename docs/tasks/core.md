# Core Tasks — V1 Stable

## Issues

- [ ] 01 — Install Godploy on Ubuntu from GHCR [easy]
- [x] 02 — Create a Project with a default production Project Instance [easy]
- [x] 03 — Create an internal application Service inside the production Project Instance [easy]
- [x] 04 — Expose a public application Service from the production Project Instance [easy]
- [ ] 05 — Show project instance switching in the dashboard [easy]
- [ ] 06 — Create a branch preview Project Instance from a production snapshot [easy]
- [ ] 07 — Create a pull request preview Project Instance from available PR candidates [easy]
- [ ] 08 — Keep open pull request candidates in SQLite and show them in the dashboard [easy]
- [ ] 09 — Update preview Project Instances from push and pull request events [easy]
- [ ] 10 — Auto-delete pull request preview Project Instances on close or merge [easy]
- [ ] 11 — Support manual delete and TTL cleanup for branch preview Project Instances [easy]
- [ ] 12 — Manage generated preview domains and manual overrides per preview Service [easy]
- [ ] 13 — Show preview instance lifecycle status separately from deployment status [easy]
- [ ] 14 — Show deployment history with rebuild and rollback actions per Service in an instance [easy]
- [x] 15 — Create a Postgres Predefined Database Service with an Internal URL [easy]
- [ ] 16 — Add Redis to the Predefined Database Service flow [easy]
- [ ] 17 — Edit a Predefined Database Service and apply changes only on redeploy [easy]
- [x] 18 — Preserve deleted database data as an Orphan Volume and show it in Storage [easy]
- [x] 19 — Reattach a compatible Orphan Volume during Predefined Database Service creation [easy]
- [ ] 20 — Warn about Orphan Volumes when deleting a Project [easy]
- [ ] 21 — Align GitHub App manifest, webhook endpoint, and public server URL behavior [hard]
- [ ] 22 — Decide the V1 critical-flow test gate and coverage order [hard]
- [ ] 23 — Decide the V1 rate-limiting policy for core routes and service exposure [hard]

## Extras V1

- [x] 01 — Global Settings: avatar picker, profile edit, password change [easy]
- [ ] 02 — Organization Settings: CRUD, org switch, delete with cascade, project transfer [easy]
- [ ] 03 — Project Instance Rename: editable name with per-project uniqueness [easy]
- [x] 04 — App Service Pause & Replicas: pause/resume, replica count inc/dec [easy]
- [ ] 05 — App Service Health Check Override: custom health check over Dockerfile [easy]
- [ ] 06 — Service Domain Settings: production domain input, preview auto-gen & custom override [easy]
- [ ] 07 — Project Deletion Warning Update: list running services per instance instead of orphan volumes [easy]
- [ ] 08 — Predefined DB Volume Size & Stop/Start: configurable volume size, stop/start without redeploy [easy]
- [ ] 09 — Orphan Volume Filters & Rename: size sort, type filter, name search, inline rename [easy]
