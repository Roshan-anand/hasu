import Image from "next/image";
import Link from "next/link";

const footerLinks = [
  { label: "GitHub", href: "https://github.com/hasu" },
  { label: "Documentation", href: "/docs" },
  { label: "Privacy", href: "#" },
];

export default function Footer() {
  return (
    <footer className="border-t border-border">
      <div className="page-rails mx-auto max-w-350">
        <div className="flex flex-col items-center gap-5 px-6 py-8 sm:flex-row sm:justify-between">
          <Link href="/" className="flex items-center gap-2">
            <Image
              src="/cow-face.png"
              alt="HASU logo"
              width={24}
              height={24}
              className="size-6"
            />
            <span className="text-sm font-semibold tracking-tight text-fg-tertiary">
              hasu
            </span>
          </Link>

          <ul className="flex items-center gap-5">
            {footerLinks.map((link) => (
              <li key={link.label}>
                <Link
                  href={link.href as any}
                  className="text-xs font-medium text-fg-tertiary transition-colors hover:text-fg-secondary"
                >
                  {link.label}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </footer>
  );
}
