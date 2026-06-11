# extras-v1 — 02: Organization Settings — CRUD & Transfer

## What to build

Organization-level settings page covering org lifecycle and project transfer.

**CRUD:**
- Rename an existing org (edit name field)
- Create a new org (user fills name, backend creates it under the current user)
- Switch active org (dropdown/selector that re-fetches project data for the selected org)
- Delete an org — shows a warning listing all projects and services that will be removed. On confirm, cascading delete in the backend (projects → instances → services → deployments → volumes).

**Transfer:**
- Inside the org delete flow (or as a separate "Transfer project" action), show the option to move a project to another org instead of losing it.
- Transfer just reassigns the `org_id` on the project record. Backend validates the target org exists and the user has access.
- If user chooses to transfer, they pick a target org from a list, then return to the delete flow.
- Org delete should succeed only after all projects are either deleted or transferred.

## Acceptance criteria

- [x] Rename, create, switch, and delete an org all work from the settings page
- [x] Delete org shows a warning with list of projects/services that will be removed
- [x] Transfer project action exists (inline in delete flow or as a separate action)
- [x] Transfer reassigns `org_id` on the project; target org validated
- [ ] Cascade deletion on org delete: projects → instances → services → deployments → volumes

## Blocked by

None — can start immediately.
