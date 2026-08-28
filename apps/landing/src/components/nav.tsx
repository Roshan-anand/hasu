"use client";

import { useEffect, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { FaGithub } from "react-icons/fa";
import { cn } from "@hasu/ui/lib/utils";
import { ModeToggle } from "./mode-toggle";

const links = [
  { href: "#features", label: "Features" },
  { href: "https://github.com/hasu", label: "GitHub" },
  { href: "/docs", label: "Docs" },
];

export default function Nav() {
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 10);
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <header
      className={cn(
        "sticky top-0 inset-x-0 z-55 h-14 border-b border-border transition-colors duration-300",
        scrolled ? "bg-background/90 backdrop-blur-xl" : "bg-background",
      )}
    >
      {/* Same rails as main so the verticals line up through the whole page */}
      <nav className="page-rails mx-auto flex h-full max-w-350 items-center justify-between px-6">
        <Link href="/" className="flex shrink-0 items-center gap-2">
          <Image
            src="/cow-face.png"
            alt="HASU logo"
            width={28}
            height={28}
            className="size-7"
          />
          <span className="text-base font-semibold tracking-tight text-foreground">
            hasu
          </span>
        </Link>

        <ul className="hidden items-center gap-6 md:flex">
          {links.map((link) => (
            <li key={link.href}>
              <Link
                href={link.href as any}
                className="text-[13px] font-medium text-fg-secondary transition-colors hover:text-foreground"
              >
                {link.label}
              </Link>
            </li>
          ))}
        </ul>

        <div className="flex gap-2 items-center">
          <Link href="https://github.com/Roshan-anand/hasu">
            <FaGithub className="size-6 text-foreground" />
          </Link>
          <ModeToggle />
        </div>
      </nav>
    </header>
  );
}
