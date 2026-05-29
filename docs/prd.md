## Problem Statement

Godploy's MVP already proves the core deployment loop for an application **Service**, but V1 still lacks the product shape and hardening needed for a reliable demo-stable release. The current gaps are centered around four areas:

- the product model needs to consistently operate as **Organization -> Project -> Service**
- application **Services** need a cleaner multi-branch model through **Service Branches**
- **Predefined Database Services** are not yet implemented as a first-class product capability
- installation, frontend completeness, runtime health visibility, and operational edge cases still need to be tightened before V1

From the user's perspective, V1 should let them install Godploy on an Ubuntu VPS, create a **Project**, deploy application **Services** with multiple **Service Branches**, create internal **Predefined Database Services** such as Postgres and Redis inside the same **Project Network**, and manage the product through a polished UI with fewer demo-breaking bugs.

## Solution

V1 will turn Godploy into a demo-stable self-hosted PaaS centered on **Projects** as the grouping and network boundary for **Services**. Each **Project** will own a **Project Network** used for private service-to-service communication through **Internal URL** values.

Application **Services** will support one or more **Service Branches**. A branch is a separately deployable instance under the same application **Service**. Branches share application-level configuration such as repository, build settings, environment, and **Exposure Mode**, while each branch has its own deployable runtime, routing state, deployment history, and branch domain.

**Predefined Database Services** will be implemented through a backend-managed **Predefined Service Template** catalog for Postgres and Redis. Users will choose a template, select an allowed **Template Version**, edit a safe subset of fields, deploy the service inside the **Project Network**, and use the generated **Internal URL** manually in their application configuration.

Godploy itself will be distributed for V1 as a single core Go server component packaged as a container image from GHCR, alongside Traefik as the ingress proxy. Installation will target Ubuntu VPS environments through an `install.sh` flow that installs Docker if missing, initializes swarm mode, pulls the required GHCR images, provisions persistence for Godploy metadata, and starts the runtime with the required Docker and Traefik configuration.

## User Stories

1. As a solo operator, I want to install Godploy on an Ubuntu VPS with one script, so that I can start using the platform quickly.
2. As a solo operator, I want Godploy to pull its runtime images from GHCR, so that I can install the product from a predictable registry.
3. As a logged-in user, I want to create a **Project** inside my **Organization**, so that I can group related **Services**.
4. As a logged-in user, I want every **Project** to provide a private **Project Network**, so that **Services** in the same **Project** can communicate internally.
5. As a logged-in user, I want to create an application **Service** inside a **Project**, so that it is isolated with the right grouping and network boundary.
6. As a logged-in user, I want to choose the initial repository branch when creating an application **Service**, so that the first **Service Branch** matches my intended deploy target.
7. As a logged-in user, I want application **Services** to have a single **Exposure Mode**, so that all of their **Service Branches** follow the same public or internal access behavior.
8. As a logged-in user, I want to create a public application **Service**, so that its **Service Branches** can be reached from the web through Traefik.
9. As a logged-in user, I want to create an internal application **Service**, so that only other **Services** on the same **Project Network** can reach it.
10. As a logged-in user, I want to add a new **Service Branch** to an application **Service**, so that I can deploy another branch as a separate instance.
11. As a logged-in user, I want a new public **Service Branch** to receive an auto-generated branch domain based on the main domain, so that preview-style routing is created automatically.
12. As a logged-in user, I want to manually edit a branch domain, so that I can override the generated routing when needed.
13. As a logged-in user, I want generated branch domains to follow the main branch when the base domain changes, while manually edited domains stay untouched, so that I keep convenience without losing explicit overrides.
14. As a logged-in user, I want to delete a non-default **Service Branch**, so that I can remove temporary or obsolete deploy targets without deleting the whole application **Service**.
15. As a logged-in user, I want to promote another **Service Branch** to be the default before deleting the old default, so that the application **Service** always has a stable default branch.
16. As a logged-in user, I want webhook-triggered deploys to rebuild only tracked branches for the matching repository and branch, so that unrelated **Service Branches** are not redeployed.
17. As a logged-in user, I want to create a Postgres **Predefined Database Service** from a built-in template, so that I can run a database without manual container setup.
18. As a logged-in user, I want to create a Redis **Predefined Database Service** from a built-in template, so that I can run internal caching or queue infrastructure easily.
19. As a logged-in user, I want template fields to be prefilled but editable, so that I can move quickly while still customizing service name, credentials, and version.
20. As a logged-in user, I want every **Predefined Database Service** to remain internal-only, so that databases are never exposed publicly by mistake.
21. As a logged-in user, I want Godploy to show me the full **Internal URL** for a **Predefined Database Service**, so that I can manually wire it into an application **Service**.
22. As a logged-in user, I want editing a **Predefined Database Service** to require an explicit redeploy before runtime changes apply, so that stateful changes do not happen unexpectedly.
23. As a logged-in user, I want to delete a **Predefined Database Service** with an optional data deletion checkbox, so that I can choose between removing runtime only or removing runtime plus data.
24. As a logged-in user, I want preserved database data to become an **Orphan Volume**, so that I can reuse it later instead of losing it.
25. As a logged-in user, I want a **Storage** area listing **Orphan Volumes**, so that I can understand which preserved volumes still exist.
26. As a logged-in user, I want a new **Predefined Database Service** to optionally attach a compatible **Orphan Volume** from the current **Project** or the unassigned pool, so that I can restore previously preserved data.
27. As a logged-in user, I want project deletion to warn me about associated **Orphan Volumes**, so that I make an explicit choice about preserving or removing detached data.
28. As a logged-in user, I want to view deployment history per **Service Branch**, so that I can understand what was deployed and when.
29. As a logged-in user, I want rebuild and rollback actions to be visible and safer around edge cases, so that I can recover or redeploy without demo-breaking surprises.
30. As a logged-in user, I want service detail pages to show both deployment progress and runtime health, so that I can distinguish build/deploy state from live container state.
31. As a logged-in user, I want runtime health to come from container health checks when present, so that the status reflects the actual running service.
32. As a logged-in user, I want runtime health to fall back to running or stopped state when no health check is defined, so that every **Service** still has useful visibility.
33. As a logged-in user, I want the UI to include loaders, confirmation dialogs, and responsive layouts for all core flows, so that the product feels complete enough for demos.
34. As a logged-in user, I want UI coverage for all backend-triggered operations, so that I do not need to leave the dashboard for normal workflows.
35. As a future team-based operator, I want Git provider connections to stay **Organization**-scoped, so that projects consume shared provider integrations consistently.

## Implementation Decisions

- V1 standardizes the domain model around **Organization -> Project -> Service**.
- A **Project** is both a grouping boundary and a **Project Network** boundary.
- Application **Services** and **Predefined Database Services** both belong to a **Project**.
- Application **Services** may contain multiple **Service Branches**.
- A **Service Branch** is the unit of separate deploy runtime, deployment history, domain assignment, and branch-level lifecycle operations.
- **Exposure Mode** belongs to the application **Service**, not to each **Service Branch**. All branches under the same application **Service** are either public or internal together.
- Public application **Services** join both the global ingress network and the **Project Network**.
- Internal application **Services** join only the **Project Network**.
- **Predefined Database Services** are always internal-only.
- The initial application **Service** creation flow includes selecting the **Project**, Git provider, repository, initial branch, and **Exposure Mode**.
- Repository/provider/build configuration stays at the application **Service** level and is inherited by all **Service Branches** under that service.
- Creating a new **Service Branch** immediately schedules a deployment job rather than creating an idle branch record.
- Branch names remain user-visible in their original form, but deploy/runtime-safe sanitized slugs are used for branch-derived domain names and swarm runtime naming.
- Public branch routing uses a generated domain pattern of `<branch_name>.<base_domain>`.
- The default branch keeps the base domain, while non-default branches get generated subdomains.
- Branch domains are editable after generation.
- Generated branch domains auto-update when the base domain changes, but manually edited branch domains remain unchanged.
- Branch deletion is supported for non-default branches.
- A default branch cannot be deleted until another branch is promoted as the default.
- Promoting a new default branch is part of V1, but remains a low-priority item within the V1 sequence.
- Webhook-triggered rebuilds match by repository and branch so that only the relevant tracked **Service Branches** are redeployed.
- Predefined services are modeled as a shared **Predefined Service Template** system rather than separate one-off product flows.
- V1 ships two predefined templates only: Postgres and Redis.
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
- Runtime status represents the live branch-level runtime state such as running, stopped, or unhealthy.
- Runtime status prefers container health checks when they exist and falls back to running or stopped when no health check is defined.
- Observability scope for V1 includes logs, deployment history, and health/status views. Broader CPU, memory, or exporter-based metrics are deferred.
- Frontend work for V1 is a first-class workstream, not a thin finishing pass. It includes complete UI coverage for backend actions, responsiveness, loaders, confirmations, and overall UX cleanup.
- Installation targets Ubuntu VPS environments only.
- The V1 installer is an `install.sh` flow.
- The installer installs Docker when missing, initializes swarm mode, pulls GHCR images for Godploy and Traefik, provisions persistence for Godploy metadata, and starts the required runtime resources.
- Godploy runs as a standalone Docker container for V1.
- Traefik runs as a swarm service for V1.
- Godploy metadata persists through dedicated storage for SQLite and Badger data.
- The Godploy dashboard itself is accessed through the server's public address on port `8080` in V1.
- Git provider integrations remain **Organization**-scoped.

### Major Modules

- **Project Topology Module**: owns the normalized **Project** lifecycle and **Project Network** rules for all **Services**.
- **Application Service Module**: owns application **Service** creation, application-level configuration, and **Exposure Mode**.
- **Service Branch Orchestrator**: owns branch creation, default-branch promotion, deletion rules, branch deploy scheduling, and branch routing metadata.
- **Branch Routing Policy Module**: owns generated branch domains, manual override tracking, public/internal routing behavior, and Traefik label generation rules.
- **Predefined Service Template Catalog**: exposes template definitions, allowed versions, safe editable fields, and runtime defaults for Postgres and Redis.
- **Predefined Database Lifecycle Module**: owns create, update, redeploy, stop, and delete behavior for predefined databases plus **Internal URL** generation.
- **Storage Module**: owns **Orphan Volume** persistence, visibility, compatibility checks, and reattachment workflows.
- **Status Aggregation Module**: owns separation and presentation of deployment status vs runtime health.
- **Installer Bootstrap Module**: owns Ubuntu installation behavior, GHCR image pulls, Docker/swarm bootstrap, and Godploy runtime startup.
- **Frontend Experience Module**: owns dashboard flows for projects, application services, service branches, predefined services, storage, confirmations, and responsive UI polish.

These modules should be kept deep where possible: the template catalog, service branch orchestration, storage/orphan-volume management, branch routing policy, and status aggregation are the clearest opportunities to encapsulate complex behavior behind stable interfaces.

## Testing Decisions

- Good tests should validate externally observable behavior rather than internal implementation details.
- The most valuable tests for V1 are the flows where product meaning and runtime behavior meet: project topology, application service creation, branch creation, webhook-triggered deploys, rebuild/rollback behavior, predefined database lifecycle, and orphan-volume reattachment.
- Deep modules should get isolated tests where possible, especially the predefined template catalog, branch routing policy, storage/orphan-volume compatibility rules, and status aggregation behavior.
- End-to-end or handler-level integration tests should continue to cover critical user-visible workflows across database state, deployment metadata, and API responses.
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
- Broad resource metrics, Prometheus-style observability, or exporter-driven monitoring.
- Upgrade automation as part of the V1 installer.
- Multi-distro installer support beyond Ubuntu.

## Further Notes

### TODO

- Review the GitHub App manifest webhook endpoint behavior against the currently implemented webhook endpoint.
- Review the Godploy public server URL/runtime scheme alignment used for GitHub redirects and webhook flows.
- Finalize the V1 testing strategy and task breakdown for critical flows.
- Finalize the rate-limiting plan for route classes and Traefik-level user-configurable service limits.
