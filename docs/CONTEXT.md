# Godploy

Godploy is a self-hosted PaaS for managing deployable services inside an organization. This glossary defines the product language used to talk about the platform's core resources.

## Language

**Organization**:
The top-level workspace that owns projects, services, and provider connections.
_Avoid_: Org account, tenant

**Project**:
A grouping inside an organization that contains related services.
_Avoid_: App, workspace, repo group

**Project Instance**:
An isolated runtime copy of a project, such as the default production instance or a temporary preview instance.
_Avoid_: Environment, workspace clone, branch runtime

**Service**:
A deployable runtime unit that belongs to exactly one project instance.
_Avoid_: App, container

**Git Source**:
The selected repository source used by an application service in an instance, such as a branch or pull request.
_Avoid_: Service branch, deploy branch, runtime branch

**Predefined Database Service**:
A service created from a built-in database template such as Postgres or Redis.
_Avoid_: Addon, plugin, managed database

**Predefined Service Template**:
A built-in service definition that provides the deploy configuration for a predefined service type.
_Avoid_: Preset, boilerplate service

**Template Version**:
The selectable base image version offered by a predefined service template.
_Avoid_: Custom image, runtime patch level

**Orphan Volume**:
A detached data volume preserved after its service is deleted.
_Avoid_: Lost volume, dangling data

**Storage**:
The product area where preserved detached volumes are listed and managed.
_Avoid_: Disk, filesystem view

**Instance Network**:
The private network shared by services in the same project instance.
_Avoid_: Project network, org network, cluster network

**Internal URL**:
The private connection URL used by a service to reach another service inside the same instance network; for predefined databases this is a full connection string.
_Avoid_: Public URL, domain

**Public Service**:
A service exposed through Traefik so it can receive external traffic.
_Avoid_: Internet app, external container

**Internal Service**:
A service reachable only from other services on the same project instance network.
_Avoid_: Hidden app, background container

**Exposure Mode**:
The access mode of an application service, either public or internal.
_Avoid_: Network mode, visibility

**Global Settings**:
User-level profile configuration including avatar, display name, email, and password.
_Avoid_: Account settings, profile page

**Service Domain**:
The domain name configured for a service, either user-entered or auto-generated for previews.
_Avoid_: Hostname, ingress domain

**Auto-generated Domain**:
A backend-generated subdomain for a preview service following the `<service>.<preview>.<base_domain>` pattern.
_Avoid_: System domain, default domain

**Custom Domain**:
A user-overridden domain for a service, replacing any auto-generated domain.
_Avoid_: Manual domain, override domain

**Organization Transfer**:
Moving a project from one organization to another by reassigning its owning organization identifier.
_Avoid_: Move project, reassign project

**Replicas**:
The number of container instances running for a service within the swarm.
_Avoid_: Instances, scale count, copies

**Pause**:
Taking an application service offline by setting its replicas to zero while preserving configuration and deployment state.
_Avoid_: Suspend, stop, disable

**Stop**:
Taking a predefined database service offline while preserving its configuration and stored data.
_Avoid_: Pause, shutdown, disable

**Health Check Override**:
A user-specified health check that supersedes any Dockerfile-defined health check for a service.
_Avoid_: Custom health check, manual health check

**Service Dependency**:
An explicit declared connection from one application **Service** to another **Service** within the same **Project Instance**, specifying an environment variable name and a target service column (e.g., `internal_url`) whose value is injected at deploy time.
_Avoid_: Service link, env injection, auto-connect

**Volume Size**:
The configurable storage capacity for a predefined database service, set at creation and editable later.
_Avoid_: Disk size, storage quota

## Relationships

- An **Organization** contains one or more **Projects**
- A **Project** belongs to exactly one **Organization**
- A **Project** contains one or more **Project Instances**
- A **Project** provides exactly one production **Project Instance**
- A **Project Instance** belongs to exactly one **Project**
- A **Project Instance** contains one or more **Services**
- A **Service** belongs to exactly one **Project Instance**
- An application **Service** uses exactly one active **Git Source** inside an instance
- A **Project Instance** provides exactly one **Instance Network** for its **Services**
- A **Predefined Database Service** is a kind of **Service**
- A **Predefined Database Service** is created from exactly one **Predefined Service Template**
- A **Predefined Database Service** selects exactly one **Template Version** from its template's allowed versions
- An **Orphan Volume** may be preserved after a **Predefined Database Service** is deleted
- An **Orphan Volume** may belong to a **Project** or be unassigned
- **Storage** contains zero or more **Orphan Volumes**
- Reattaching a preserved volume removes it from **Storage** and assigns it back to a compatible **Predefined Database Service**
- A **Public Service** joins the **Instance Network** and the global ingress network
- An **Internal Service** joins only the **Instance Network**
- A **Service** may use an **Internal URL** to communicate with another **Service** in the same **Project Instance**
- An application **Service** may have zero or more **Service Dependencies** on other **Services** in the same **Project Instance**
- A **Service Dependency** resolves a target **Service** column value into an environment variable at deploy time
- A **Service** keeps a configurable **Volume Size** when it is a **Predefined Database Service**
- A **Service** may have a **Service Domain** that is either **Auto-generated** or **Custom**
- A **Service** may be **Paused** by setting its **Replicas** to zero
- A **Predefined Database Service** may be **Stopped** separately from being deleted
- A **Service** may carry an optional **Health Check Override** that supersedes the Dockerfile health check
- A **Project** may be transferred from one **Organization** to another via **Organization Transfer**
- A user has one **Global Settings** profile containing avatar, name, email, and password

## Example dialogue

> **Dev:** "If I create a Postgres database for this backend, does it belong to the organization or the project?"
> **Domain expert:** "It belongs to a **Project Instance**, and the other **Services** in that same instance can reach it through its **Internal URL**."

> **Dev:** "Can an application service be private too, or are only databases private?"
> **Domain expert:** "Any **Service** can be internal-only; a **Public Service** is the one that also gets external ingress."

> **Dev:** "How much can the user change when creating Postgres?"
> **Domain expert:** "They choose from a **Predefined Service Template**, then edit safe fields like name, credentials, and the allowed **Template Version**."

> **Dev:** "If I create a preview from a pull request, is that just another branch under the same service?"
> **Domain expert:** "No, it is a separate **Project Instance** with its own cloned **Services** and its own **Instance Network**."

> **Dev:** "How does one app service know what code to run in production versus preview?"
> **Domain expert:** "Each application **Service** points to one **Git Source** inside its current instance."

> **Dev:** "What happens to database data when I delete the service but keep the data?"
> **Domain expert:** "The data becomes an **Orphan Volume** and can later be managed from **Storage** or attached again to a compatible database service."

> **Dev:** "Can I temporarily take an app offline without deleting it?"
> **Domain expert:** "Yes, you **Pause** the **Service** which sets its **Replicas** to zero while preserving everything else."

> **Dev:** "How do I move a project to a different organization?"
> **Domain expert:** "Use **Organization Transfer** — it simply reassigns the owning organization on the **Project** record."

> **Dev:** "Does the preview service get a domain automatically?"
> **Domain expert:** "Yes, the backend generates an **Auto-generated Domain** when the preview is created. You can override it with a **Custom Domain** in the service settings."

> **Dev:** "How do I connect my backend app to a Postgres database without manually copying the connection string?"
> **Domain expert:** "Use the **Service Dependency** feature — in your backend **Service** settings, click 'Connect Service', select the Postgres **Service**, choose the `internal_url` column, and name the environment variable `DATABASE_URL`. The system injects the current value at every deploy, and automatically rewrites it when you create a preview instance."

> **Dev:** "What happens if I change the Postgres password after connecting it to my app?"
> **Domain expert:** "The system updates the stored **Service Dependency** value in the background. Your app picks up the new connection string on its next deploy — no manual reconnection needed."

> **Dev:** "Can I still set environment variables manually if I don't want to use the connect feature?"
> **Domain expert:** "Yes, the **Service Dependency** feature is fully optional. You can still copy the **Internal URL** and paste it into your **Service** environment variables manually, just like before."

## Flagged ambiguities

- `service` was previously discussed as belonging directly to an **Organization**; resolved: a **Service** belongs to a **Project Instance**, and that instance belongs to a **Project**.
- `application` was previously used to imply public access; resolved: an application may be either a **Public Service** or an **Internal Service**.
- `network` was used to describe service visibility; resolved: use **Exposure Mode** for public vs internal, and **Instance Network** for the private network itself.
- `branch` was previously used as the runtime deployment boundary; resolved: use **Project Instance** for runtime isolation and **Git Source** for repository branch or pull request selection.
- `account` was previously used interchangeably with user profile; resolved: use **Global Settings** for the user-level configuration and profile data.
- `suspend` was used to describe taking a service offline; resolved: use **Pause** for application services and **Stop** for predefined database services.
- `domain override` was used loosely; resolved: a **Service Domain** that is user-entered is a **Custom Domain**, distinct from the **Auto-generated Domain** for previews.
- `service connection` was used ambiguously to mean both manual env copy-paste and automated dependency injection; resolved: manual copy-paste is just environment configuration, while **Service Dependency** is the explicit declared connection that the system manages and resolves at deploy time.
