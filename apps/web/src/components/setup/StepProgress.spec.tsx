import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { StepProgress } from './StepProgress';

const STEPS = [
  { id: 'welcome', title: '歡迎' },
  { id: 'qbittorrent', title: 'qBittorrent' },
  { id: 'media-folder', title: '媒體資料夾' },
  { id: 'api-keys', title: 'API 金鑰' },
  { id: 'complete', title: '完成' },
];

describe('StepProgress', () => {
  it('renders the correct number of dots', () => {
    render(<StepProgress steps={STEPS} currentStep={0} />);
    expect(screen.getByTestId('step-dot-welcome')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-qbittorrent')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-media-folder')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-api-keys')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-complete')).toBeInTheDocument();
  });

  it('marks current and previous steps as active (blue)', () => {
    const { container } = render(<StepProgress steps={STEPS} currentStep={2} />);
    const dots = container.querySelectorAll('[data-testid^="step-dot-"]');
    // Steps 0, 1, 2 should have bg-[var(--accent-primary)]
    expect(dots[0]).toHaveClass('bg-[var(--accent-primary)]');
    expect(dots[1]).toHaveClass('bg-[var(--accent-primary)]');
    expect(dots[2]).toHaveClass('bg-[var(--accent-primary)]');
    // Steps 3, 4 should have bg-[var(--bg-tertiary)]
    expect(dots[3]).toHaveClass('bg-[var(--bg-tertiary)]');
    expect(dots[4]).toHaveClass('bg-[var(--bg-tertiary)]');
  });

  it('marks only first dot active at step 0', () => {
    const { container } = render(<StepProgress steps={STEPS} currentStep={0} />);
    const dots = container.querySelectorAll('[data-testid^="step-dot-"]');
    expect(dots[0]).toHaveClass('bg-[var(--accent-primary)]');
    expect(dots[1]).toHaveClass('bg-[var(--bg-tertiary)]');
  });

  it('marks all dots active at last step', () => {
    const { container } = render(<StepProgress steps={STEPS} currentStep={4} />);
    const dots = container.querySelectorAll('[data-testid^="step-dot-"]');
    for (const dot of dots) {
      expect(dot).toHaveClass('bg-[var(--accent-primary)]');
    }
  });

  it('has correct aria labels', () => {
    render(<StepProgress steps={STEPS} currentStep={2} />);
    expect(screen.getByLabelText('媒體資料夾 (目前)')).toBeInTheDocument();
    expect(screen.getByLabelText('歡迎')).toBeInTheDocument();
  });
});
