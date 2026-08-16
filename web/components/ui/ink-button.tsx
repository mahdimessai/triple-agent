import type { ReactNode } from "react";

export function InkButton({ children, variant, className = "", onClick, disabled }: { children: ReactNode; variant?: "orange"; className?: string; onClick?: () => void; disabled?: boolean }) {
  return <button className={`ta-ink-button px-5 ${className}`} data-variant={variant} onClick={onClick} disabled={disabled ? true : undefined} type="button">{children}</button>;
}
