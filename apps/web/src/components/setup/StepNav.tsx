// Design ref: ux-design.pen Screen N2-D (CP7AX) · N3-D (TyjL0) · N4-D (D990CP) · N5-D (CWh3E)
// The button row every wizard step ends with. The order is the design's, and it
// is fixed: 上一步 · 跳過 · 下一步. The primary action takes the remaining width and
// sits LAST, so it stays under the thumb on a phone and never moves depending on
// whether this particular step happens to be skippable.
import { Button } from '../ui/Button';

interface StepNavProps {
  onNext: () => void;
  onBack?: () => void;
  onSkip?: () => void;
  nextLabel?: string;
  nextTestId?: string;
  nextDisabled?: boolean;
  backDisabled?: boolean;
}

export function StepNav({
  onNext,
  onBack,
  onSkip,
  nextLabel = '下一步',
  nextTestId = 'next-button',
  nextDisabled = false,
  backDisabled = false,
}: StepNavProps) {
  return (
    <div className="flex gap-3">
      {onBack && (
        <Button
          type="button"
          variant="secondary"
          onClick={onBack}
          disabled={backDisabled}
          className="h-11 px-4 font-semibold"
          data-testid="back-button"
        >
          上一步
        </Button>
      )}
      {onSkip && (
        <Button
          type="button"
          variant="ghost"
          onClick={onSkip}
          className="h-11 px-3 text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          data-testid="skip-button"
        >
          跳過
        </Button>
      )}
      <Button
        type="button"
        onClick={onNext}
        disabled={nextDisabled}
        className="h-11 flex-1 font-semibold"
        data-testid={nextTestId}
      >
        {nextLabel}
      </Button>
    </div>
  );
}
