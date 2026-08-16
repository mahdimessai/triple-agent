import type { ReactNode } from "react";
import styles from "./mock-workbench.module.css";

export function MockGameShell({ children }: { children: ReactNode }) {
  return (
    <main className={styles.viewport}>
      <div className={styles.device}>
        <header className={styles.gameHeader}>
          <div className={styles.brand}>
            <span className={styles.brandEyebrow}>TRIPLE AGENT · PRODUCTION MOCKS</span>
            <p className={styles.brandTitle}>Reference workbench</p>
          </div>
          <nav className={styles.headerTabs} aria-label="Mock workbench references">
            <a className={styles.headerTab} href="#screen-index">
              SCREENS
            </a>
            <a className={styles.headerTab} href="#fixture-projection">
              FIXTURES
            </a>
          </nav>
        </header>
        <div className={styles.activityRail} role="status" aria-live="polite">
          <span className={styles.statusLabel}>MOCK STATE</span>
          <span>FIXTURE ONLY · LIVE PLAY PREVIEW</span>
        </div>
        {children}
      </div>
    </main>
  );
}
