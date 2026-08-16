import type { MockScreenDefinition, MockScreenGroup, MockScreenId } from "./mock-screen-registry";
import { mockScreenGroups } from "./mock-screen-registry";
import { MockArtStamp, privacyLabel } from "./MockScreenFrame";
import styles from "./mock-workbench.module.css";

const groupDescriptions: Record<MockScreenGroup, string> = {
  FLOW: "Entry, setup, agency briefing, and the handoff into operations.",
  OPERATIONS: "Waiting, private payloads, and public explanation states for an operation.",
  VOTING: "Discussion history and the private accusation handoff before results.",
  RESULTS: "A staged reveal from tally to rematch.",
};

export function MockScreenPicker({
  selectedId,
  screens,
  onSelect,
}: {
  selectedId: MockScreenId;
  screens: readonly MockScreenDefinition[];
  onSelect: (id: MockScreenId) => void;
}) {
  return (
    <aside className={styles.picker} id="screen-index" aria-label="Production mock screen picker">
      <div className={styles.pickerIntro}>
        <p className={styles.eyebrow}>ADR-005 · SINGLE ROUTE</p>
        <h1 className={styles.pickerTitle}>Screen index</h1>
        <p className={styles.pickerCopy}>
          Choose a deterministic presentation. The picker is local workbench state and never navigates the multiplayer room.
        </p>
      </div>
      {mockScreenGroups.map((group) => {
        const groupScreens = screens.filter((screen) => screen.group === group);
        return (
          <section className={styles.group} key={group} aria-labelledby={`mock-group-${group.toLowerCase()}`}>
            <div className={styles.groupHeading}>
              <h2 className={styles.groupTitle} id={`mock-group-${group.toLowerCase()}`}>
                {group}
              </h2>
              <span className={`${styles.groupCount} ${styles.tileMeta}`}>{groupScreens.length} TILES</span>
            </div>
            <p className={styles.smallCopy}>{groupDescriptions[group]}</p>
            <div className={styles.tileGrid}>
              {groupScreens.map((screen) => (
                <button
                  className={styles.tile}
                  data-selected={screen.id === selectedId}
                  data-status={screen.status}
                  key={screen.id}
                  onClick={() => onSelect(screen.id as MockScreenId)}
                  type="button"
                  aria-label={`Preview ${screen.title}, ${privacyLabel(screen.privacy)}`}
                  aria-pressed={screen.id === selectedId}
                >
                  <span className={styles.tileTop} aria-hidden="true">
                    <span className={styles.numberLabel}>{screen.number}</span>
                    <MockArtStamp artName={screen.artName} alt="" className={styles.tileArt} />
                  </span>
                  <span>
                    <span className={styles.tileLabel}>{screen.label}</span>
                    <span className={styles.tileMeta}>{screen.phase ?? "PRESENTATION"}</span>
                  </span>
                  <span className={styles.tileDescription}>{screen.description}</span>
                  <span className={styles.tileFooter}>
                    <span>{privacyLabel(screen.privacy)}</span>
                    <span>{screen.status === "planned" ? "LOCKED" : "READY"}</span>
                  </span>
                </button>
              ))}
            </div>
          </section>
        );
      })}
    </aside>
  );
}
