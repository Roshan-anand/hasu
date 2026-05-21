# MVP RoadMap

#### Goal : User able to install and authenticate into app. create project and service (Predefined). connect Github and pull build and deploy the repo. manage env ssl and domain.

## Stories : As as user ...

- [ ] i want to install Godploy on my vps in one cmd.
- [x] i want to register/login using email and password.
- [x] i want to CRUD organizations.
- [x] i want to connect to Github.
- [x] i want to select a repo and branch.
- [x] i want to CRUD services.
- [x] i want to setup Application service.
- [ ] i want to setup DB services (PSQL, MongoDB).
- [x] i want to deploy the service.
- [x] i want to view the build logs.
- [x] i want to view the service logs.
- [x] i want to set ssl certs for the service.
- [x] i want to set custom domain for the service.

## Tasks :

- **Setup**
  - [ ] Install script (.sh) to setup Godploy and Traefik.
  - [ ] Uninstall script (.sh) to remove Godploy and Traefik.
  - [x] Setup cloudflare tunnel for local webhook testing.
  - [x] Setup Traefik for domain routing and ssl management.
  - [x] Setup docker swarm mode.

- **Authentication**
  - [x] User Registration and Login (email/password).
  - [x] JWT-based authentication for API access.
  - [ ] Password reset functionality.
  - [ ] invite team members via email.
  - [ ] RBAC

- **ORG/project**
  - [x] CRUD for Org.

- **Services**
  - [x] CRUD for Services.
  - [x] Deploy, Stop, Rebuild service.
  - [ ] Predefined service templates (PSQL, MongoDB).
  - [x] Application Service template (for user code deployments).
  - [x] Workers logs streaming.
  - [x] Pull Job
  - [x] Build Job
  - [x] Deploy Job
  - [x] Service logs streaming.
  - [x] Environment variable management.
  - [ ] Build secrets management.
  - [x] SSL certificate management (Let's Encrypt integration).
  - [x] Custom domain management (Traefik integration).

- **OCI Image Builder**
  - [x] Build images using Dockerfile.
  - [ ] Build images using Nixpacks.
  - [ ] Build images using Buildpacks.

- **Github Integration**
  - [x] installation of GitHub App.
  - [x] Fetch the repositories and branches.
  - [x] WebHooks to trigger auto-deploy on push.
  - [ ] Build context management - Cleaning up old build artifacts.
