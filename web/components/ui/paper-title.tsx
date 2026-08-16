import type { ReactNode } from "react";

export function PaperTitle({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <h2 className={`ta-paper ta-display ta-angle-left px-4 py-3 text-center text-[clamp(1.25rem,4vw,2.3rem)] ${className}`}>{children}</h2>;
}
