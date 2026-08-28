interface Feature {
  heading: string;
  description: string;
  label: string;
  mockupKind: "form" | "list" | "graph" | "database";
}

const features: Feature[] = [
  {
    heading: "Simple Form-Fill Deploy",
    description:
      "Connect your GitHub repo, pick a branch, and deploy. No YAML, no Dockerfiles required.",
    label: "New Service",
    mockupKind: "form",
  },
  {
    heading: "Instant Preview Instances",
    description:
      "Every pull request gets its own isolated environment. Full service snapshot, fresh databases, auto-cleanup.",
    label: "Preview Instances",
    mockupKind: "list",
  },
  {
    heading: "Scale Up and Down",
    description:
      "Add services, wire dependencies, and grow your project. HASU handles the runtime boundaries.",
    label: "Service Graph",
    mockupKind: "graph",
  },
  {
    heading: "Predefined Services",
    description:
      "One-click Postgres and Redis. Internal-only, auto-wired, with optional data preservation.",
    label: "Add Database",
    mockupKind: "database",
  },
];

function MockupFrame({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="w-full overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-[10px] font-medium text-fg-tertiary">{label}</span>
        <div className="flex gap-1">
          <span className="size-1 rounded-full bg-fg-tertiary/30" />
          <span className="size-1 rounded-full bg-fg-tertiary/30" />
          <span className="size-1 rounded-full bg-fg-tertiary/30" />
        </div>
      </div>
      <div className="p-3">{children}</div>
    </div>
  );
}

function FormMockup() {
  return (
    <div className="space-y-2">
      <div className="h-1.5 w-14 rounded-sm bg-fill" />
      <div className="h-6 w-full rounded border border-border bg-background" />
      <div className="mt-2 h-1.5 w-16 rounded-sm bg-fill" />
      <div className="h-6 w-full rounded border border-border bg-background" />
      <div className="mt-2 h-1.5 w-12 rounded-sm bg-fill" />
      <div className="h-6 w-full rounded border border-border bg-background" />
      <div className="mt-3 flex gap-2">
        <div className="h-6 flex-1 rounded border border-primary/25 bg-primary/15" />
        <div className="h-6 flex-1 rounded border border-border bg-background" />
      </div>
    </div>
  );
}

function ListMockup() {
  return (
    <div className="space-y-0.5">
      {[
        {
          name: "api-service",
          status: "LIVE",
          color: "text-green-600 dark:text-green-400/60",
          dot: "bg-green-600 dark:bg-green-400/60",
        },
        {
          name: "web-frontend",
          status: "",
          color: "",
          dot: "bg-fg-tertiary/40",
        },
        {
          name: "worker-queue",
          status: "BUILDING",
          color: "text-amber-600 dark:text-amber-400/60",
          dot: "bg-amber-600 dark:bg-amber-400/60",
        },
        {
          name: "cron-jobs",
          status: "",
          color: "",
          dot: "bg-fg-tertiary/40",
        },
      ].map((item, i) => (
        <div key={i} className="flex items-center gap-2.5 rounded px-2 py-1.5">
          <span className={`size-1.5 rounded-full ${item.dot}`} />
          <span className="flex-1 text-[11px] text-muted-foreground">
            {item.name}
          </span>
          {item.status && (
            <span className={`text-[9px] font-medium ${item.color}`}>
              {item.status}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

function GraphMockup() {
  return (
    <div className="flex items-center justify-center gap-4 py-3">
      {["API", "Web", "Worker"].map((label, i) => (
        <div key={label} className="flex flex-col items-center gap-2">
          <div
            className={`flex size-9 items-center justify-center rounded-full border ${
              i === 0
                ? "border-primary/25 bg-primary/8"
                : "border-border bg-background"
            }`}
          >
            <div
              className={`size-3 rounded-sm ${i === 0 ? "bg-primary/40" : "bg-fill"}`}
            />
          </div>
          <span className="text-[10px] text-fg-tertiary">{label}</span>
        </div>
      ))}
    </div>
  );
}

function DatabaseMockup() {
  return (
    <div className="space-y-1.5">
      {[
        {
          badge: "PG",
          badgeColor: "bg-[#4169e1]/25 text-[#4169e1] dark:text-[#6b9fff]",
          label: "postgres-16",
          connected: true,
        },
        {
          badge: "R",
          badgeColor: "bg-[#dc382d]/25 text-[#dc382d] dark:text-[#ff6b6b]",
          label: "redis-7",
          connected: false,
        },
      ].map((db, i) => (
        <div
          key={i}
          className="flex items-center gap-2.5 rounded border border-border bg-background px-2.5 py-2"
        >
          <span
            className={`inline-flex size-5 items-center justify-center rounded-sm text-[9px] font-bold ${db.badgeColor}`}
          >
            {db.badge}
          </span>
          <span className="flex-1 text-[11px] text-muted-foreground">
            {db.label}
          </span>
          <span
            className={`flex h-4 w-10 items-center justify-center rounded text-[9px] font-medium ${
              db.connected
                ? "border border-primary/25 bg-primary/15 text-primary"
                : "border border-border bg-background text-fg-tertiary"
            }`}
          >
            {db.connected ? "wired" : "idle"}
          </span>
        </div>
      ))}
      <div className="mt-3 flex items-center gap-2 rounded border border-dashed border-border px-2.5 py-2">
        <span className="text-xs text-fg-tertiary">+</span>
        <span className="text-[10px] text-fg-tertiary">Add service</span>
      </div>
    </div>
  );
}

function MockupContent({ kind }: { kind: Feature["mockupKind"] }) {
  switch (kind) {
    case "form":
      return <FormMockup />;
    case "list":
      return <ListMockup />;
    case "graph":
      return <GraphMockup />;
    case "database":
      return <DatabaseMockup />;
  }
}

export default function Features() {
  return (
    <section
      id="features"
      className="flex flex-col gap-[var(--frame-gutter)] py-[var(--frame-gutter)]"
    >
      {features.map((feature, i) => {
        const isEven = i % 2 === 0;
        return (
          <div key={feature.heading} className="box-frame">
            <div className="flex flex-col gap-10 px-6 py-[clamp(2.5rem,5vw,4rem)] md:px-8 lg:flex-row lg:items-center">
              <div
                className="flex-1 space-y-3"
                style={{ order: isEven ? 0 : 1 }}
              >
                <span className="text-[11px] font-medium uppercase tracking-[0.06em] text-fg-tertiary">
                  {i + 1 < 9 ? `0${i + 1}` : i + 1}&nbsp;&nbsp;·&nbsp;&nbsp;
                  {feature.label}
                </span>
                <h2
                  className="font-semibold leading-[1.15] tracking-[-0.01em] text-foreground text-[clamp(1.5rem,3vw,2.25rem)]"
                  style={{ textWrap: "balance" }}
                >
                  {feature.heading}
                </h2>
                <p className="max-w-sm text-[0.938rem] leading-relaxed text-muted-foreground">
                  {feature.description}
                </p>
              </div>

              <div className="flex-1" style={{ order: isEven ? 1 : 0 }}>
                <MockupFrame label={feature.label}>
                  <MockupContent kind={feature.mockupKind} />
                </MockupFrame>
              </div>
            </div>
          </div>
        );
      })}
    </section>
  );
}
