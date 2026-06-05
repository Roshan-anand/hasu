# V2 Follow-up

## Promote a preview Project Instance to production

A user action that takes an existing preview **Project Instance** and replaces the current production instance with it. After promotion, the preview identity becomes the new production identity for that project.

- the preview snapshot has to be revalidated against current production expectations
- production credentials, secrets, and external integrations need a clean handoff
- historical deployment records need to be preserved through the swap
- the operation must be reversible or at least resumable if it fails halfway


### Open V2 questions

- atomic swap versus rolling promotion per service
- whether to keep the prior production instance as a backup or delete it
- how user-edited preview domains transition into the production domain model
- what happens to in-flight preview webhooks during promotion

### Likely V2 user story

> As a logged-in user, I want to promote a preview **Project Instance** to production, so that a tested preview becomes the new stable runtime without rebuilding it from scratch.

---

## Seeded preview data and snapshot-based preview database initialization

### What it is

The ability to populate preview database and cache services with realistic data instead of starting from a fresh empty state. Two flavors were discussed:

- **Seeded data**: ship a known dataset, fixture, or migration set that runs against preview stateful services at creation time
- **Snapshot-based init**: copy the current production database into the preview database at creation time, optionally with masking rules

### Why it was deferred

Both flavors carry real risk:

- seeded data can drift from production schema and cause preview-only false positives
- snapshot-based init can leak real production data into previews
- both add cost in time and storage on every preview creation
- both require a clear masking or redaction story before they are safe to enable by default

V1 ships with fresh empty stateful services in every preview to keep the safety story simple. Anything that pulls production data into a preview will need additional review.

### Open V2 questions

- default masking policy for snapshot-based init
- whether snapshot init is opt-in per project, per service, or per preview
- size and time budget for snapshot init
- how seeding interacts with monorepo change targeting
- whether seeding is a template-defined default or a user-authored script

### Likely V2 user story

> As a logged-in user, I want to seed my preview database with masked production data, so that I can test real-world flows without polluting production.

---

## Richer preview-instance topology editing workflows

### What it is

The ability to add, remove, or reshape services inside a preview instance without going back to the production instance. Examples:

- add a new experimental service to a preview
- remove a sibling service from a preview because it is irrelevant to the test
- clone an extra worker into a preview
- run a one-off migration inside the preview only

### Why it was deferred

V1 deliberately keeps preview creation as a full production clone and forbids divergent topology edits inside a preview. That keeps the model honest:

- previews stay reproducible from a known snapshot
- preview cleanup stays total and predictable
- users cannot accidentally grow a preview into a long-lived second production

Adding preview-only topology edits in V1 would have required a parallel diffing model, separate cleanup paths, and a way to reconcile preview-only services back into production if they ever want to land. All of that is real product work, not a small extra.

### What V1 must preserve so V2 can add it

- preview services are full independent records, not references back to production
- deletion inside a preview is allowed in V1 even though new service creation inside a preview is not
- preview instance lifecycle status is already modeled separately from per-service status
- preview instance has its own dedicated network and storage

### Open V2 questions

- diff model for preview-only topology changes
- whether preview-only services can be promoted into production service definitions
- how preview-only service changes interact with snapshot pinning
- whether preview-only services trigger a separate webhook rebuild path
- how preview cleanup handles preview-only orphaned services

### Likely V2 user story

> As a logged-in user, I want to add an experimental service to a preview **Project Instance**, so that I can iterate on a feature branch without touching the production topology.

---