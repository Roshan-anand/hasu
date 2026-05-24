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

**Predefined Database Service**:
A service created from a built-in database template such as Postgres or Redis.
_Avoid_: Addon, plugin, managed database

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

## Relationships

- An **Organization** contains one or more **Projects**
- A **Project** belongs to exactly one **Organization**
- A **Project** contains one or more **Services**
- A **Service** belongs to exactly one **Project**
- A **Project** provides exactly one **Project Network** for its **Services**
- A **Predefined Database Service** is a kind of **Service**
- A **Public Service** joins the **Project Network** and the global ingress network
- An **Internal Service** joins only the **Project Network**
- A **Service** may use an **Internal URL** to communicate with another **Service** in the same **Project**

## Example dialogue

> **Dev:** "If I create a Postgres database for this backend, does it belong to the organization or the project?"
> **Domain expert:** "It belongs to the **Project**, and the other **Services** in that **Project** can reach it through its **Internal URL**."

> **Dev:** "Can an application service be private too, or are only databases private?"
> **Domain expert:** "Any **Service** can be internal-only; a **Public Service** is the one that also gets external ingress."

## Flagged ambiguities

- `service` was previously discussed as belonging directly to an **Organization**; resolved: a **Service** belongs to a **Project**, and a **Project** belongs to an **Organization**.
- `application` was previously used to imply public access; resolved: an application may be either a **Public Service** or an **Internal Service**.
