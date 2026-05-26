# Godploy

Godploy is a self-hosted PaaS for managing deployable services inside an organization. This glossary defines the product language used to talk about the platform's core resources.

## Language

**Organization**:
The top-level workspace that owns projects, services, and provider connections.
_Avoid_: Org account, tenant

**Project**:
A grouping inside an organization that contains related services.
_Avoid_: App, workspace, repo group

**Service**:
A deployable runtime unit that belongs to exactly one project.
_Avoid_: App, container

**Service Branch**:
A separately deployable branch instance of an application service.
_Avoid_: Environment, clone, duplicate service

**Predefined Database Service**:
A service created from a built-in database template such as Postgres or Redis.
_Avoid_: Addon, plugin, managed database

**Predefined Service Template**:
A built-in service definition that provides the deploy configuration for a predefined service type.
_Avoid_: Preset, boilerplate service

**Template Version**:
The selectable base image version offered by a predefined service template.
_Avoid_: Custom image, runtime patch level

**Project Network**:
The private network shared by services in the same project.
_Avoid_: Org network, cluster network

**Internal URL**:
The private connection address a service uses to reach another service inside the same project.
_Avoid_: Public URL, domain

**Public Service**:
A service exposed through Traefik so it can receive external traffic.
_Avoid_: Internet app, external container

**Internal Service**:
A service reachable only from other services on the same project network.
_Avoid_: Hidden app, background container

**Exposure Mode**:
The access mode of a service, either public or internal.
_Avoid_: Network mode, visibility

## Relationships

- An **Organization** contains one or more **Projects**
- A **Project** belongs to exactly one **Organization**
- A **Project** contains one or more **Services**
- A **Service** belongs to exactly one **Project**
- An application **Service** may contain one or more **Service Branches**
- A **Project** provides exactly one **Project Network** for its **Services**
- A **Predefined Database Service** is a kind of **Service**
- A **Predefined Database Service** is created from exactly one **Predefined Service Template**
- A **Predefined Database Service** selects exactly one **Template Version** from its template's allowed versions
- A **Public Service** joins the **Project Network** and the global ingress network
- An **Internal Service** joins only the **Project Network**
- A **Service** has exactly one **Exposure Mode** at a time
- A **Service** may use an **Internal URL** to communicate with another **Service** in the same **Project**

## Example dialogue

> **Dev:** "If I create a Postgres database for this backend, does it belong to the organization or the project?"
> **Domain expert:** "It belongs to the **Project**, and the other **Services** in that **Project** can reach it through its **Internal URL**."

> **Dev:** "Can an application service be private too, or are only databases private?"
> **Domain expert:** "Any **Service** can be internal-only; a **Public Service** is the one that also gets external ingress."

> **Dev:** "How much can the user change when creating Postgres?"
> **Domain expert:** "They choose from a **Predefined Service Template**, then edit safe fields like name, credentials, and the allowed **Template Version**."

> **Dev:** "If I deploy another git branch, is that a new service?"
> **Domain expert:** "No, it is another **Service Branch** under the same application **Service**."

## Flagged ambiguities

- `service` was previously discussed as belonging directly to an **Organization**; resolved: a **Service** belongs to a **Project**, and a **Project** belongs to an **Organization**.
- `application` was previously used to imply public access; resolved: an application may be either a **Public Service** or an **Internal Service**.
- `network` was used to describe service visibility; resolved: use **Exposure Mode** for public vs internal, and **Project Network** for the private network itself.
