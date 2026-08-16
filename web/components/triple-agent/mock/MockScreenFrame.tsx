import Image from "next/image";
import type { ReactNode } from "react";
import { art, type ArtName } from "../asset-registry";
import type { MockScreenDefinition, MockPrivacy } from "./mock-screen-registry";
import styles from "./mock-workbench.module.css";

export function MockArtStamp({
  artName,
  alt,
  className,
  priority = false,
}: {
  artName: ArtName;
  alt: string;
  className?: string;
  priority?: boolean;
}) {
  const item = art[artName];

  return (
    <Image
      src={item.src}
      alt={alt}
      width={item.width}
      height={item.height}
      priority={priority}
      className={className}
    />
  );
}

export function MockPaperPanel({
  children,
  className = "",
  angle,
  dataFaction,
}: {
  children: ReactNode;
  className?: string;
  angle?: "left" | "right";
  dataFaction?: "SERVICE" | "VIRUS";
}) {
  return (
    <section className={`${styles.paperPanel} ${className}`} data-angle={angle} data-faction={dataFaction}>
      {children}
    </section>
  );
}

export function MockInkButton({
  children,
  className = "",
  variant,
  onClick,
}: {
  children: ReactNode;
  className?: string;
  variant?: "orange";
  onClick?: () => void;
}) {
  return (
    <button className={`${styles.inkButton} ${className}`} data-variant={variant} onClick={onClick} type="button">
      {children}
    </button>
  );
}

export function MockStatusLabel({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <span className={`${styles.statusLabel} ${className}`}>{children}</span>;
}

export function privacyLabel(privacy: MockPrivacy) {
  switch (privacy) {
    case "private":
      return "PRIVATE TO YOU";
    case "mixed":
      return "PUBLIC + PRIVATE";
    default:
      return "PUBLIC ROOM";
  }
}

export function MockScreenFrame({
  definition,
  children,
}: {
  definition: MockScreenDefinition;
  children: ReactNode;
}) {
  return (
    <article className={styles.screenFrame} data-screen-id={definition.id} aria-labelledby={`${definition.id}-title`}>
      <header className={styles.screenHeader}>
        <div>
          <span className={styles.phaseLabel}>{definition.phase ?? "FIXTURE VIEW"}</span>
          <h3 className={styles.screenTitle} id={`${definition.id}-title`}>
            {definition.title}
          </h3>
        </div>
        <span className={styles.privacyLabel}>{privacyLabel(definition.privacy)}</span>
      </header>
      <div className={styles.screenBody}>{children}</div>
      <footer className={styles.screenFooter}>
        <p className={styles.statusLabel}>DETERMINISTIC MOCK · NO SERVER CONNECTION</p>
        <p>Private states replace the original pass-and-hold gate.</p>
      </footer>
    </article>
  );
}
