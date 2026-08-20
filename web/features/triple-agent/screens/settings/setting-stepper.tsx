export function SettingStepper({
  label,
  value,
  hint,
  disabled = false,
  canDecrease,
  canIncrease,
  onDecrease,
  onIncrease,
}: {
  label: string;
  value: string;
  hint: string;
  disabled?: boolean;
  canDecrease: boolean;
  canIncrease: boolean;
  onDecrease(): void;
  onIncrease(): void;
}) {
  return (
    <div className="ta-stepper">
      <p className="ta-condensed text-xs tracking-[0.16em] uppercase text-black/65">{label}</p>
      <div className="mt-2 flex items-stretch gap-2">
        <button
          className="ta-stepper-arrow"
          type="button"
          aria-label={`Decrease ${label.toLowerCase()}`}
          disabled={disabled || !canDecrease}
          onClick={onDecrease}
        >
          &#9664;
        </button>
        <output className="ta-stepper-value ta-display">{value}</output>
        <button
          className="ta-stepper-arrow"
          type="button"
          aria-label={`Increase ${label.toLowerCase()}`}
          disabled={disabled || !canIncrease}
          onClick={onIncrease}
        >
          &#9654;
        </button>
      </div>
      <p className="ta-sans mt-2 text-xs leading-snug text-black/70">{hint}</p>
    </div>
  );
}
