const steps = [
  {
    number: "01",
    title: "Install.",
    description:
      "SSH into any Ubuntu box, run one script, and HASU is live on :8080. Docker and Traefik are configured automatically.",
  },
  {
    number: "02",
    title: "Connect.",
    description:
      "Link your GitHub repo, pick a branch, deploy your first service. No YAML, no Dockerfiles — just a form and a button.",
  },
  {
    number: "03",
    title: "Preview.",
    description:
      "Every pull request gets a full isolated environment. Fresh databases, cloned services, auto-cleanup on merge.",
  },
];

export default function HowItWorks() {
  return (
    <section>
      <div className="box-frame">
        <div className="flex items-center gap-3 border-b border-border px-6 py-5 md:px-8">
          <span className="text-[11px] font-medium uppercase tracking-[0.06em] text-primary">
            How it works
          </span>
          <span className="h-px flex-1 bg-border" />
        </div>

        <div className="grid gap-px bg-border lg:grid-cols-3">
          {steps.map((step) => (
            <div
              key={step.number}
              className="flex flex-col gap-3 bg-background p-8"
            >
              <span className="text-[0.75rem] font-medium tracking-[0.06em] text-fg-tertiary/50">
                {step.number}
              </span>
              <h3 className="text-base font-semibold tracking-[-0.01em] text-foreground">
                {step.title}
              </h3>
              <p className="text-[0.875rem] leading-relaxed text-muted-foreground">
                {step.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
