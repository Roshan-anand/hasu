"use client";

import Link from "next/link";
import InstallCommand from "./install-command";

export default function CTA() {
  return (
    <section className="py-[var(--frame-gutter)]">
      <div className="box-frame">
        <div className="grid gap-10 px-6 py-[clamp(2.5rem,5vw,4rem)] md:px-8 lg:grid-cols-2 lg:items-center">
          <div className="space-y-6">
            <span className="text-[11px] font-medium uppercase tracking-[0.06em] text-primary">
              Get started
            </span>

            <h2
              className="font-semibold leading-[1.15] tracking-[-0.01em] text-foreground text-[clamp(1.5rem,3vw,2.25rem)]"
              style={{ textWrap: "balance" }}
            >
              Own your infrastructure.
              <br />
              Start in one command.
            </h2>

            <InstallCommand variant="dark" />

            <Link
              href={"#" as any}
              className="inline-flex items-center justify-center rounded-[0.5rem] border border-border px-5 py-2.5 text-sm font-medium text-muted-foreground transition-all hover:border-stroke-active hover:text-foreground"
            >
              Watch demo
            </Link>
          </div>

          <div className="relative aspect-[16/10] overflow-hidden rounded-lg border border-border bg-card">
            <div
              className="absolute inset-0 bg-cover bg-center opacity-15"
              style={{ backgroundImage: "url('/bg-cow.jpeg')" }}
            />
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="flex size-14 items-center justify-center rounded-full border border-border bg-background/80 backdrop-blur-sm">
                <svg
                  width="16"
                  height="18"
                  viewBox="0 0 16 18"
                  fill="none"
                  className="ml-0.5"
                >
                  <path
                    d="M15 9L0.75 17.2265L0.75 0.773501L15 9Z"
                    className="fill-foreground"
                    fillOpacity="0.6"
                  />
                </svg>
              </div>
            </div>
            <span className="absolute bottom-3 right-4 text-[10px] text-fg-tertiary">
              [DEMO VIDEO]
            </span>
          </div>
        </div>
      </div>
    </section>
  );
}
