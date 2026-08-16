"use client";

import { useState } from "react";
import { renderMockScreen } from "@/features/triple-agent/mock/fixture-adapters";
import { mockFixture, type MockFixture } from "./mock-fixtures";
import { getMockScreen, mockScreenRegistry, type MockScreenId } from "./mock-screen-registry";
import { MockGameShell } from "./MockGameShell";
import { MockScreenFrame, MockStatusLabel } from "./MockScreenFrame";
import { MockScreenPicker } from "./MockScreenPicker";
import styles from "./mock-workbench.module.css";

function ActivityPanel({ fixture }: { fixture: MockFixture }) {
  return (
    <aside className={styles.activityPanel} id="fixture-projection" aria-label="Public mock activity rail">
      <div className={styles.activityHeading}>
        <div>
          <p className={styles.eyebrow}>PUBLIC PROJECTION</p>
          <h2 className={styles.activityTitle}>Activity rail</h2>
        </div>
        <span className={styles.activityPulse} aria-label="Projection connected" />
      </div>
      <p className={styles.activityCopy}>A compact public channel for room-level state. Private payloads never appear here.</p>
      <div className={styles.activityList}>
        {fixture.activity.map((entry) => (
          <div className={styles.activityItem} data-tone={entry.tone} key={entry.label}>
            <span className={styles.activityLabel}>{entry.label}</span>
            <span>{entry.detail}</span>
          </div>
        ))}
      </div>
      <div className={styles.activityFooter}>
        <MockStatusLabel>FIXTURE ONLY</MockStatusLabel>
        <p>Live play replaces this rail with server projection and connection feedback.</p>
      </div>
    </aside>
  );
}

export function MockWorkbench() {
  const [selectedId, setSelectedId] = useState<MockScreenId>(mockScreenRegistry[0].id);
  const selectedScreen = getMockScreen(selectedId);

  return (
    <MockGameShell>
      <div className={styles.workbench}>
        <MockScreenPicker selectedId={selectedId} screens={mockScreenRegistry} onSelect={setSelectedId} />
        <section className={styles.previewColumn} aria-label="Selected mock screen preview">
          <div className={styles.previewHeading}>
            <div>
              <p className={styles.eyebrow}>SELECTED FIXTURE · {selectedScreen.number}</p>
              <h2>{selectedScreen.label}</h2>
              <p>{selectedScreen.description}</p>
            </div>
            <span className={`${styles.previewIndex} ${styles.numberLabel}`}>{selectedScreen.group}</span>
          </div>
          <div className={styles.previewCanvas}>
            <MockScreenFrame definition={selectedScreen}>
              {renderMockScreen(selectedScreen, mockFixture)}
            </MockScreenFrame>
          </div>
        </section>
        <ActivityPanel fixture={mockFixture} />
      </div>
    </MockGameShell>
  );
}
