import Image from "next/image";
import Link from "next/link";
import { FaGithub, FaDiscord } from "react-icons/fa";
import InstallCommand from "@/components/install-command";
import { Button } from "@hasu/ui/components/button";

export default function Hero() {
  return (
    <section className="flex flex-col pt-[var(--frame-gutter)]">
      {/* Copy panel — full box with rules extending into gutters */}
      <div className="box-frame px-6 py-10 md:px-10 md:py-14 relative flex items-center">
        <section>
          <p className="mb-3 text-[11px] font-medium uppercase tracking-[0.06em] text-fg-tertiary">
            Self-hosted PaaS
          </p>

          <h1
            className="font-bold tracking-[-0.03em] text-foreground text-[clamp(2.25rem,5vw,4.5rem)] leading-[1.05]"
            style={{ textWrap: "balance" }}
          >
            Deploy anything. Own everything.
          </h1>

          <p className="mt-4 max-w-2xl text-[0.938rem] leading-relaxed text-muted-foreground md:text-base">
            An open-source & self-hostable alternative to Vercel, Netlify &
            Railway for easily deploying websites, databases, web applications.
          </p>
        </section>
        <section className="flex flex-col gap-4">
          <InstallCommand />
          <div className="flex items-center gap-3">
            <Button>
              <Link
                href="https://github.com/Roshan-anand/hasu"
                className="flex items-center gap-1"
                target="_blank"
                rel="noopener noreferrer"
              >
                <FaGithub className="size-4" />
                <p>GitHub</p>
              </Link>
            </Button>
              <Button className={"bg-[#5865F2] text-white"}>
              <Link
                href="#"
                className="flex items-center gap-1"
                target="_blank"
                rel="noopener noreferrer"
              >
                <FaDiscord className="size-4" />
                <p>Discord</p>
              </Link>
            </Button>
          </div>
        </section>

        <span className="bg-primary size-5 w-8 absolute bottom-0 right-0" />
        <span className="bg-primary size-5 w-8 absolute bottom-0 left-0" />
        {/*<span className="bg-primary size-20 absolute top-0 right-0" />*/}
      </div>

      {/* Visual panel — same boxy frame as copy */}
      <div className="mx-auto w-[92%] p-4 border-l border-r relative">
        
        <span className="bg-primary size-5 w-8 absolute -top-px z-50 -left-8" />
        <span className="bg-primary size-5 w-8 absolute -top-px z-50 -right-8" />
        
        <div className="box-frame relative flex min-h-[280px] justify-center overflow-hidden p-10 w-full md:min-h-[400px] md:h-160 lg:h-270 mx-auto">
          <Image
            height={700}
            width={900}
            src="/cow-bg.png"
            alt=""
            aria-hidden
            className="pointer-events-none absolute inset-0 size-full object-cover object-bottom opacity-60 blur-[0.9px]"
          />
          <div className="relative z-20 size-full h-fit overflow-hidden rounded-md">
            <Image
              height={900}
              width={1200}
              src="/hasu-dashboard.png"
              alt="hasu dashboard"
              className="size-full rounded-md object-contain object-center"
            />
          </div>
        </div>
      </div>
    </section>
  );
}
