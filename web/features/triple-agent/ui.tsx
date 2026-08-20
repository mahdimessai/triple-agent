import Image from "next/image";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { art, type ArtName } from "./assets";

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

export function PaperTitle({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <h2 className={`ta-paper ta-display ta-angle-left px-4 py-3 text-center text-[clamp(1.25rem,4vw,2.3rem)] ${className}`}>
      {children}
    </h2>
  );
}

export type ArtStampProps = {
  artName: ArtName;
  alt?: string;
  className?: string;
  priority?: boolean;
  sizes?: string;
};

export function ArtStamp({ artName, alt = "", className = "", priority = false, sizes = "(max-width: 640px) 50vw, 320px" }: ArtStampProps) {
  const item = art[artName];
  return (
    <Image
      src={item.src}
      alt={alt}
      width={item.width}
      height={item.height}
      priority={priority}
      loading={priority ? "eager" : "lazy"}
      sizes={sizes}
      className={className}
    />
  );
}
