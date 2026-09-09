"use client";

import { useState } from "react";
import { Copy, Check } from "lucide-react";
import { cn } from "@hasu/ui/lib/utils";

// const COMMAND = "curl -sSL https://raw.githubusercontent.com/hasu/install.sh | sh";
const COMMAND = "curl -sSL https://hasu.rshn.cloud/install.sh | sh";

export default function InstallCommand({
  className,
  variant = "light",
}: {
  className?: string;
  variant?: "light" | "dark";
}) {
  const [copied, setCopied] = useState(false);
  const dark = variant === "dark";

  const handleCopy = async () => {
    await navigator.clipboard.writeText(COMMAND);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div
      className={cn(
        "flex w-full max-w-lg items-center gap-0 overflow-hidden border",
        dark
          ? "border-fg-inverted/15 bg-fg-inverted/5"
          : "border-border bg-card",
        className,
      )}
    >
      <code
        className={cn(
          "flex-1 truncate px-4 py-3 text-sm select-all",
          dark ? "text-fg-inverted" : "text-foreground",
        )}
      >
        {COMMAND}
      </code>
      <button
        onClick={handleCopy}
        className={cn(
          "flex shrink-0 items-center gap-1.5 border-l px-4 py-3 text-sm font-medium transition-colors",
          dark
            ? "border-fg-inverted/15 text-fg-inverted/70 hover:bg-fg-inverted/10 hover:text-fg-inverted"
            : "border-border text-fg-secondary hover:bg-bg-hover hover:text-foreground",
        )}
        aria-label={copied ? "Copied" : "Copy install command"}
      >
        {copied ? (
          <Check className="size-4 text-accent-positive-highlight" />
        ) : (
          <Copy className="size-4" />
        )}
        <span className="hidden sm:inline">{copied ? "Copied" : "Copy"}</span>
      </button>
    </div>
  );
}
