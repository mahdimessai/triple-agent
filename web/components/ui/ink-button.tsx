import type { ReactNode } from "react";

export function InkButton({ children, variant, className = "", onClick, disabled, loading = false, loadingLabel }: { children: ReactNode; variant?: "orange"; className?: string; onClick?: () => void; disabled?: boolean; loading?: boolean; loadingLabel?: string }) {
  const unavailable = disabled || loading;
  return (
    <button
      className={`ta-ink-button px-5 ${className}`}
      data-variant={variant}
      onClick={onClick}
      disabled={unavailable ? true : undefined}
      aria-busy={loading ? true : undefined}
      type="button"
    >
      {loading ? <span className="ta-button-loading"><span className="ta-button-spinner" aria-hidden="true" />{loadingLabel ?? children}</span> : children}
    </button>
  );
}
