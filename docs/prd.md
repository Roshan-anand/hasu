## Problem Statement

Godploy's MVP already proves the core deployment loop for an application **Service**, but the current multi-branch approach is flawed for realistic preview testing. Branch deploys currently behave like extra runtimes inside the same project shape, which means a user testing a branch or pull request can still depend on production-connected sibling services and production-like shared topology.

The V1 gaps are now centered around four areas:

- the product model needs to consistently operate as **Organization -> Project -> Project Instance -> Service**
- preview deployments need full-instance isolation instead of runtime-level branch clones under one application service
- **Predefined Database Services** still need to exist as a first-class product capability inside each instance
- installation, frontend completeness, runtime visibility, and GitHub automation still need to be tightened before a reliable demo-stable release

From the user's perspective, V1 should let them install Godploy on an Ubuntu VPS, create a **Project**, get a default production **Project Instance**, create isolated preview **Project Instances** from branches or pull requests, run internal **Predefined Database Services** such as Postgres and Redis inside each instance, and manage the product through a polished UI with fewer demo-breaking surprises.

## Solution

V1 will turn Godploy into a demo-stable self-hosted PaaS centered on **Projects** as the long-lived grouping boundary and **Project Instances** as the runtime boundary. Every **Project** owns exactly one production **Project Instance** by default, and may also own multiple preview **Project Instances** created from a selected branch or pull request. Each **Project Instance** owns its own private network, cloned service set, runtime state, deployment history, and routing.

Application **Services** no longer use runtime-level **Service Branches** as the preview model. Instead, each application **Service** inside an instance points to a **Git Source**. A **Git Source** may be the inherited production branch, a manually selected branch, or a pull request. Preview creation snapshots the current production instance, clones all services into a new preview instance, rebuilds only the services selected by the chosen **Git Source** rules, and reuses pinned ready images for the unchanged application services.

**Predefined Database Services** continue to be implemented through a backend-managed **Predefined Service Template** catalog for Postgres and Redis, but they are now instance-scoped. Preview instances always get fresh isolated stateful services and fresh volumes rather than sharing production data.

Godploy itself will be distributed for V1 as a single core Go server component packaged as a container image from GHCR, alongside Traefik as the ingress proxy. Installation will target Ubuntu VPS environments through an `install.sh` flow that installs Docker if missing, initializes swarm mode, pulls the required GHCR images, provisions persistence for Godploy metadata, and starts the runtime with the required Docker and Traefik configuration.

## User Stories

1. As a solo operator, I want to install Godploy on an Ubuntu VPS with one script, so that I can start using the platform quickly.
2. As a solo operator, I want Godploy to pull its runtime images from GHCR, so that I can install the product from a predictable registry.
3. As a logged-in user, I want to create a **Project** inside my **Organization**, so that I can group related deployable systems.
4. As a logged-in user, I want each new **Project** to automatically create one production **Project Instance**, so that I can start from a stable default runtime.
5. As a logged-in user, I want every **Project Instance** to provide its own private instance network, so that services inside one instance are isolated from services in other instances.
6. As a logged-in user, I want to create an application **Service** inside the production **Project Instance**, so that it becomes part of the project's primary runtime.
7. As a logged-in user, I want to choose the initial repository branch when creating an application **Service**, so that its first production **Git Source** matches my intended deploy target.
8. As a logged-in user, I want application **Services** to keep one **Exposure Mode**, so that each cloned copy of that service inside an instance inherits the same public or internal behavior.
9. As a logged-in user, I want to create a public application **Service**, so that its instance-local runtime can be reached from the web through Traefik.
10. As a logged-in user, I want to create an internal application **Service**, so that only other services inside the same **Project Instance** can reach it.
11. As a logged-in user, I want to create a preview **Project Instance** from a branch, so that I can test a branch in an isolated copy of the production topology.
12. As a logged-in user, I want to create a preview **Project Instance** from a pull request, so that I can test proposed changes without risking the production instance.
13. As a logged-in user, I want preview creation to clone the whole production instance snapshot, so that sibling services and stateful dependencies are isolated too.
14. As a logged-in user, I want only the services affected by the selected branch or pull request to rebuild from source, so that preview creation is faster and still faithful to the changed code.
15. As a logged-in user, I want unchanged application services in a preview to reuse pinned ready images from production, so that the preview stays reproducible without unnecessary rebuilds.
16. As a logged-in user, I want preview database and cache services to start with fresh isolated state, so that testing cannot corrupt production data.
17. As a logged-in user, I want the preview flow to offer copying production env and secrets with an explicit warning and inline editing, so that I can balance convenience and safety.
18. As a logged-in user, I want public preview services to receive generated preview domains, so that I can access isolated preview instances externally.
19. As a logged-in user, I want to manually edit a preview service domain, so that I can override generated routing when needed.
20. As a logged-in user, I want GitHub webhook events to keep preview pull request metadata in SQLite while the PR is open, so that the dashboard can surface available previews.
21. As a logged-in user, I want pull request and push events to redeploy only the matching services for the matching instance, repository, and watched paths, so that unrelated services are not rebuilt.
22. As a logged-in user, I want monorepo watch paths to decide which services switch to a pull request or branch source, so that one repo can back multiple services safely.
23. As a logged-in user, I want a preview instance to be updated in place when new commits arrive for its tracked branch or pull request, so that I keep one stable preview URL and history per source.
24. As a logged-in user, I want a pull request preview to be removed automatically when the PR is closed or merged, so that temporary runtime state does not accumulate.
25. As a logged-in user, I want branch previews to support manual deletion and TTL-based cleanup, so that temporary environments can be reclaimed automatically when desired.
26. As a logged-in user, I want preview deletion to clean up all cloned services, deployments, networks, and volumes, so that removing a preview fully removes its isolated runtime.
27. As a logged-in user, I want to create a Postgres **Predefined Database Service** from a built-in template, so that I can run a database without manual container setup.
28. As a logged-in user, I want to create a Redis **Predefined Database Service** from a built-in template, so that I can run internal caching or queue infrastructure easily.
29. As a logged-in user, I want template fields to be prefilled but editable, so that I can move quickly while still customizing service name, credentials, and version.
30. As a logged-in user, I want every **Predefined Database Service** to remain internal-only, so that databases are never exposed publicly by mistake.
31. As a logged-in user, I want Godploy to show me the full **Internal URL** for a **Predefined Database Service**, so that I can manually wire it into an application **Service**.
32. As a logged-in user, I want editing a **Predefined Database Service** to require an explicit redeploy before runtime changes apply, so that stateful changes do not happen unexpectedly.
33. As a logged-in user, I want to delete a **Predefined Database Service** with an optional data deletion checkbox, so that I can choose between removing runtime only or removing runtime plus data.
34. As a logged-in user, I want preserved database data to become an **Orphan Volume**, so that I can reuse it later instead of losing it.
35. As a logged-in user, I want a **Storage** area listing **Orphan Volumes**, so that I can understand which preserved volumes still exist.
36. As a logged-in user, I want a new **Predefined Database Service** to optionally attach a compatible **Orphan Volume** from the current **Project** or the unassigned pool, so that I can restore previously preserved data.
37. As a logged-in user, I want project deletion to warn me about associated **Orphan Volumes**, so that I make an explicit choice about preserving or removing detached data.
38. As a logged-in user, I want to view deployment history per service inside an instance, so that I can understand what was deployed and when.
39. As a logged-in user, I want rebuild and rollback actions to be visible and safer around edge cases, so that I can recover or redeploy without demo-breaking surprises.
40. As a logged-in user, I want service detail pages to show both deployment progress and runtime health, so that I can distinguish build/deploy state from live container state.
41. As a logged-in user, I want runtime health to come from container health checks when present, so that the status reflects the actual running service.
42. As a logged-in user, I want runtime health to fall back to running or stopped state when no health check is defined, so that every **Service** still has useful visibility.
43. As a logged-in user, I want the project dashboard to let me switch between production and preview instances before viewing services, so that the instance boundary is obvious in the UI.
44. As a logged-in user, I want the UI to include loaders, confirmation dialogs, and responsive layouts for all core flows, so that the product feels complete enough for demos.
45. As a logged-in user, I want UI coverage for all backend-triggered operations, so that I do not need to leave the dashboard for normal workflows.
46. As a future team-based operator, I want Git provider connections to stay **Organization**-scoped, so that projects consume shared provider integrations consistently.

## Implementation Decisions

- V1 standardizes the domain model around **Organization -> Project -> Project Instance -> Service**.
- Every **Project** owns exactly one production **Project Instance**.
- Preview **Project Instances** are explicit records rather than implicit branch runtimes.
- A **Project Instance** is the runtime boundary for services, routing, deployments, and private networking.
- A **Project** remains the long-lived grouping boundary and ownership boundary.
- Application **Services** and **Predefined Database Services** are both instance-scoped at runtime.
- Production is the source topology used when creating preview instances.
- Creating a preview instance snapshots the current production instance and clones all services into the preview.
- Preview services are fully separate runtime records and keep backlinks to their production-origin services for traceability.
- Preview resources are fully owned by the preview instance with no shared service, deployment, network, or volume ownership.
- Existing previews remain pinned to the production snapshot taken at creation time; they do not drift forward when production later changes.
- Production env and secrets are copied into a preview at creation time when chosen; they are never live-linked.
- Application **Services** no longer use runtime-level **Service Branches** as the preview model.
- **Git Source** is the runtime-facing source selection for an application service inside an instance.
- A **Git Source** may point to the production branch, a manually selected branch, or a pull request.
- The initial production application **Service** creation flow includes selecting the **Project**, Git provider, repository, initial branch, and **Exposure Mode**.
- Repository, provider, build configuration, watch path, and **Exposure Mode** stay at the application-service definition level and are copied into preview instances.
- Monorepo change targeting is determined by `repo_id` plus `watch_path` matching.
- Preview creation rebuilds only the services selected by the chosen branch or pull request rules.
- Unchanged application services in a preview reuse the exact pinned ready images from the production snapshot.
- Preview creation fails if any required production service lacks a usable ready deployment image.
- Preview creation fails if the production topology contains unsupported service types.
- Preview stateful services always get fresh isolated volumes in V1; production data is never mounted into previews.
- Public and internal behavior is inherited from the production service definition into each cloned service.
- Public preview routing uses a generated domain pattern of `<service>.<preview>.<base_domain>`.
- Source names remain user-visible in their original form, but deploy-safe sanitized slugs are used for runtime naming and generated domains.
- Preview service domains are editable and any edit is local to that preview service only.
- Manual branch previews may be created from any branch even if no webhook record exists for that branch.
- At most one active preview exists per `project + pull_request_number`.
- At most one active preview exists per `project + repo_id + branch_name` for manual branch previews.
- Pull request previews are limited to pull requests accessible through the installed repository app flow in V1.
- GitHub remains the source of truth for branches and pull requests.
- SQLite stores open pull request metadata surfaced by webhooks so the dashboard can show available preview candidates.
- Pull request rows are added on PR open or reopen and removed on PR close or merge.
- Webhook-driven update behavior is scoped by instance, repository, source, and watched paths.
- Pull request updates re-evaluate changed files against all relevant services in the instance and may expand which services rebuild from PR source.
- Once a service in a preview switches to a selected branch or pull request source, it stays on that preview snapshot's chosen source until the preview is deleted or updated intentionally.
- Pull request previews are updated in place on new commits rather than creating replacement preview instances.
- Auto-cleanup on PR close or merge should be executed asynchronously through workers.
- Branch previews support both manual deletion and TTL-based cleanup.
- Preview deletion removes all cloned services, deployments, networks, and volumes.
- Preview cleanup failures should leave the preview instance in a recoverable async state such as deleting or error rather than disappearing silently.
- Preview instances have their own lifecycle status separate from per-service deployment status.
- V1 allows deleting services inside preview instances even though preview creation itself always starts from a full production clone.
- V1 does not support adding new services only inside a preview instance through a separate topology workflow.
- **Predefined Database Services** are always internal-only.
- V1 ships two predefined templates only: Postgres and Redis.
- Predefined services are modeled as a shared **Predefined Service Template** system rather than separate one-off product flows.
- The backend owns the predefined template catalog and sends template metadata to the client.
- Each predefined template includes image selection rules, allowed **Template Versions**, default configuration values, volume behavior, internal connection information, and operational defaults.
- User-editable predefined fields for V1 include service name, credentials, logical database name where relevant, and allowed **Template Version**.
- Version selection is constrained to curated template versions rather than arbitrary image input.
- Default service names for predefined services are generated from project and template context with a random suffix, while remaining editable by the user.
- **Predefined Database Services** do not support swarm-level replicas or database-replication topology in V1.
- Editing predefined database settings updates saved configuration and requires an explicit redeploy before runtime changes take effect.
- The stored **Internal URL** for a **Predefined Database Service** is a full private connection string, not only a host or port.
- When credentials or logical database settings change, the generated **Internal URL** must be recomputed from the new stored values.
- Predefined database attachment to application **Services** remains manual in V1. Godploy shows the **Internal URL**, and the user places it into service environment settings themselves.
- Deleting a predefined database service includes an optional data-purge choice.
- If data is preserved, it becomes an **Orphan Volume** instead of remaining attached to the deleted service.
- **Orphan Volumes** belong either to a specific **Project** or to an unassigned pool when the parent **Project** is later removed while preserving data.
- Reattaching a preserved volume removes it from **Storage** and assigns it to the newly created compatible **Predefined Database Service**.
- Volume reattachment compatibility is enforced by predefined service type, with only a warning for risky version mismatches.
- Project deletion warns about associated **Orphan Volumes** and lets the user choose whether preserved data should remain detached or be removed.
- Runtime status is modeled separately from deployment status.
- Deployment status continues to represent deploy workflow state such as queued, building, ready, error, inactive, and pruned.
- Runtime status represents the live service-level runtime state such as running, stopped, or unhealthy.
- Runtime status prefers container health checks when they exist and falls back to running or stopped when no health check is defined.
- Observability scope for V1 includes logs, deployment history, and health/status views. Broader CPU, memory, or exporter-based metrics are deferred.
- Frontend work for V1 is a first-class workstream, not a thin finishing pass. It includes complete UI coverage for backend actions, instance switching, responsiveness, loaders, confirmations, and overall UX cleanup.
- Installation targets Ubuntu VPS environments only.
- The V1 installer is an `install.sh` flow.
- The installer installs Docker when missing, initializes swarm mode, pulls GHCR images for Godploy and Traefik, provisions persistence for Godploy metadata, and starts the required runtime resources.
- Godploy runs as a standalone Docker container for V1.
- Traefik runs as a swarm service for V1.
- Godploy metadata persists through dedicated storage for SQLite and Badger data.
- The Godploy dashboard itself is accessed through the server's public address on port `8080` in V1.
- Git provider integrations remain **Organization**-scoped.

### Major Modules

- **Project Topology Module**: owns the normalized **Project** lifecycle and production-instance bootstrap rules.
- **Project Instance Orchestrator**: owns production and preview instance lifecycle, snapshot creation, lifecycle status, TTL cleanup, and async deletion behavior.
- **Application Service Module**: owns application service creation, application-level configuration, watch paths, and **Exposure Mode**.
- **Git Source Resolution Module**: owns source selection for branches and pull requests plus repo and watch-path matching behavior.
- **Preview Routing Policy Module**: owns generated preview domains, manual overrides, public/internal routing behavior, and Traefik label generation rules.
- **Predefined Service Template Catalog**: exposes template definitions, allowed versions, safe editable fields, and runtime defaults for Postgres and Redis.
- **Predefined Database Lifecycle Module**: owns create, update, redeploy, stop, and delete behavior for predefined databases plus **Internal URL** generation.
- **Storage Module**: owns **Orphan Volume** persistence, visibility, compatibility checks, and reattachment workflows.
- **Status Aggregation Module**: owns separation and presentation of instance status, deployment status, and runtime health.
- **GitHub Event Intake Module**: owns webhook verification, open-PR cache updates, and deploy-target expansion rules.
- **Installer Bootstrap Module**: owns Ubuntu installation behavior, GHCR image pulls, Docker and swarm bootstrap, and Godploy runtime startup.
- **Frontend Experience Module**: owns dashboard flows for projects, instance switching, preview creation, service details, predefined services, storage, confirmations, and responsive UI polish.

These modules should be kept deep where possible: the project-instance orchestrator, Git source resolution, preview routing policy, template catalog, storage or orphan-volume management, and status aggregation are the clearest opportunities to encapsulate complex behavior behind stable interfaces.

## Testing Decisions

- Good tests should validate externally observable behavior rather than internal implementation details.
- The most valuable tests for V1 are the flows where product meaning and runtime behavior meet: production instance bootstrap, application service creation, preview instance creation, webhook-driven preview updates, rebuild and rollback behavior, predefined database lifecycle, and orphan-volume reattachment.
- Deep modules should get isolated tests where possible, especially the project-instance orchestrator, Git source resolution rules, preview routing policy, storage or orphan-volume compatibility rules, and status aggregation behavior.
- End-to-end or handler-level integration tests should continue to cover critical user-visible workflows across database state, deployment metadata, instance lifecycle, and API responses.
- Prior art already exists in the codebase through backend integration tests around authentication, organization behavior, and project behavior, plus small focused unit tests for security and auth utilities.
- V1 planning does not require the full test suite to be resolved immediately, but the testing conversation must remain an explicit follow-up workstream rather than an implicit future cleanup.

## Out of Scope

- Teams, invites, and RBAC beyond the current organization model.
- Password reset flows.
- Git providers beyond the existing GitHub-centered flow.
- Buildpacks and Nixpacks support.
- Automatic injection of predefined database credentials into application service environment values.
- Public exposure for predefined databases.
- Swarm replicas or true replication topology for Postgres or Redis.
- Additional predefined templates beyond Postgres and Redis.
- Custom user-defined predefined-service templates.
- Automated seeded preview data or production-data snapshot cloning into previews.
- Promote preview instance to production.
- Dedicated V1 topology-edit workflows that add new services only inside a preview instance.
- Broad resource metrics, Prometheus-style observability, or exporter-driven monitoring.
- Upgrade automation as part of the V1 installer.
- Multi-distro installer support beyond Ubuntu.

## Further Notes

### TODO

- Review the GitHub App manifest webhook endpoint behavior against the currently implemented webhook endpoint.
- Review the Godploy public server URL and runtime scheme alignment used for GitHub redirects and webhook flows.
- Finalize the V1 testing strategy and task breakdown for critical flows.
- Finalize the rate-limiting plan for route classes and Traefik-level user-configurable service limits.
