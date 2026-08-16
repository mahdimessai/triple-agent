import type { ReactNode } from "react";

export function InkButton({ children, variant, className = "", onClick, disabled = false }: { children: ReactNode; variant?: "orange"; className?: string; onClick?: () => void; disabled?: boolean }) {
  return <button className={`ta-ink-button px-5 ${className}`} data-variant={variant} onClick={onClick} disabled={disabled} type="button">{children}</button>;
}
