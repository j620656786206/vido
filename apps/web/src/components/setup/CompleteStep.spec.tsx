import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { CompleteStep } from './CompleteStep';
import type { StepProps } from './SetupWizard';

function makeProps(overrides?: Partial<StepProps>): StepProps {
  return {
    data: { language: 'zh-TW', mediaFolderPath: '/media' },
    onUpdate: vi.fn(),
    onNext: vi.fn(),
    onBack: vi.fn(),
    isFirst: false,
    isLast: true,
    ...overrides,
  };
}

describe('CompleteStep', () => {
  it('renders completion heading', () => {
    render(<CompleteStep {...makeProps()} />);
    expect(screen.getByText('設定完成！')).toBeInTheDocument();
  });

  it('shows summary with correct language value', () => {
    render(<CompleteStep {...makeProps()} />);
    expect(screen.getByText('zh-TW')).toBeInTheDocument();
  });

  it('shows "未設定" for missing optional fields', () => {
    render(<CompleteStep {...makeProps({ data: { language: 'en' } })} />);
    // qBT, media folder, TMDb, AI all show 未設定
    const items = screen.getAllByText('未設定');
    expect(items.length).toBeGreaterThanOrEqual(3);
  });

  it('shows "已設定" for TMDb when key exists', () => {
    render(<CompleteStep {...makeProps({ data: { language: 'en', tmdbApiKey: 'key123' } })} />);
    expect(screen.getByText('已設定')).toBeInTheDocument();
  });

  it('shows qBT URL in summary when provided', () => {
    render(
      <CompleteStep {...makeProps({ data: { language: 'en', qbtUrl: 'http://qbt:8080' } })} />
    );
    expect(screen.getByText('http://qbt:8080')).toBeInTheDocument();
  });

  it('shows AI provider name in summary', () => {
    render(<CompleteStep {...makeProps({ data: { language: 'en', aiProvider: 'gemini' } })} />);
    expect(screen.getByText('gemini')).toBeInTheDocument();
  });

  it('has finish button (not next)', () => {
    render(<CompleteStep {...makeProps()} />);
    expect(screen.getByTestId('finish-button')).toBeInTheDocument();
    expect(screen.getByTestId('finish-button')).toHaveTextContent('完成設定');
  });

  it('shows submitting state', () => {
    render(<CompleteStep {...makeProps({ isSubmitting: true })} />);
    expect(screen.getByTestId('finish-button')).toHaveTextContent('儲存中...');
    expect(screen.getByTestId('finish-button')).toBeDisabled();
  });

  it('back button is disabled during submit', () => {
    render(<CompleteStep {...makeProps({ isSubmitting: true })} />);
    expect(screen.getByTestId('back-button')).toBeDisabled();
  });

  it('calls onNext (finish handler) when clicked', () => {
    const onNext = vi.fn();
    render(<CompleteStep {...makeProps({ onNext })} />);
    fireEvent.click(screen.getByTestId('finish-button'));
    expect(onNext).toHaveBeenCalledTimes(1);
  });

  it('calls onBack when back clicked', () => {
    const onBack = vi.fn();
    render(<CompleteStep {...makeProps({ onBack })} />);
    fireEvent.click(screen.getByTestId('back-button'));
    expect(onBack).toHaveBeenCalledTimes(1);
  });
});
