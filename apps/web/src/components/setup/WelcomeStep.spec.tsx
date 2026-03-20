import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { WelcomeStep } from './WelcomeStep';
import type { StepProps } from './SetupWizard';

function makeProps(overrides?: Partial<StepProps>): StepProps {
  return {
    data: { language: 'zh-TW' },
    onUpdate: vi.fn(),
    onNext: vi.fn(),
    onBack: vi.fn(),
    isFirst: true,
    isLast: false,
    ...overrides,
  };
}

describe('WelcomeStep', () => {
  it('renders welcome heading and description', () => {
    render(<WelcomeStep {...makeProps()} />);
    expect(screen.getByText('歡迎使用 Vido')).toBeInTheDocument();
    expect(screen.getByText(/NAS 媒體管理工具/)).toBeInTheDocument();
  });

  it('renders language select with default zh-TW', () => {
    render(<WelcomeStep {...makeProps()} />);
    const select = screen.getByTestId('language-select') as HTMLSelectElement;
    expect(select.value).toBe('zh-TW');
  });

  it('shows all language options', () => {
    render(<WelcomeStep {...makeProps()} />);
    expect(screen.getByText('繁體中文')).toBeInTheDocument();
    expect(screen.getByText('English')).toBeInTheDocument();
    expect(screen.getByText('日本語')).toBeInTheDocument();
  });

  it('calls onUpdate when language changes', () => {
    const onUpdate = vi.fn();
    render(<WelcomeStep {...makeProps({ onUpdate })} />);
    fireEvent.change(screen.getByTestId('language-select'), { target: { value: 'en' } });
    expect(onUpdate).toHaveBeenCalledWith({ language: 'en' });
  });

  it('calls onNext when Next button clicked', () => {
    const onNext = vi.fn();
    render(<WelcomeStep {...makeProps({ onNext })} />);
    fireEvent.click(screen.getByTestId('next-button'));
    expect(onNext).toHaveBeenCalledTimes(1);
  });

  it('has accessible label for language select', () => {
    render(<WelcomeStep {...makeProps()} />);
    expect(screen.getByLabelText('選擇語言')).toBeInTheDocument();
  });
});
