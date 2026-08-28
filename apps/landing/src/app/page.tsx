import Nav from "@/components/nav";
import Hero from "@/components/hero";
import HowItWorks from "@/components/how-it-works";
import Features from "@/components/features";
import CTA from "@/components/cta";
import Footer from "@/components/footer";

export default function Home() {
  return (
    <>
      <Nav />
      {/*
        Continuous outer rails (border-x) + frame gutters so nested
        .box-frame horizontal rules can stick out past their verticals.
      */}
      <main className="page-rails mx-auto max-w-350">
        <div className="frame-gutter">
          <Hero />
          <HowItWorks />
          <Features />
          <CTA />
        </div>
      </main>
      <Footer />
    </>
  );
}
