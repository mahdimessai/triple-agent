import type { ButtonHTMLAttributes, ReactNode } from "react";

export type InkButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "type"> & {
  children: ReactNode;
  variant?: "orange";
  busy?: boolean;
  busyLabel?: string;
};

export function InkButton({ children, variant, className = "", busy = false, busyLabel, disabled = false, ...props }: InkButtonProps) {
  const isDisabled = Boolean(disabled || busy);
  return (
    <button
      {...props}
      className={`ta-ink-button px-5 ${className}`}
      data-variant={variant}
      disabled={isDisabled}
      aria-busy={busy ? "true" : undefined}
      type="button"
    >
      {busy ? (
        <span className="ta-button-loading">
          <span className="ta-button-spinner" aria-hidden="true" />
          {busyLabel ?? children}
        </span>
      ) : children}
    </button>
  );
}
