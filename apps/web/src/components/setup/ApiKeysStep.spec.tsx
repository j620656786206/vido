import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ApiKeysStep } from './ApiKeysStep';
import type { StepProps } from './SetupWizard';

function makeProps(overrides?: Partial<StepProps>): StepProps {
  return {
    data: {},
    onUpdate: vi.fn(),
    onNext: vi.fn(),
    onBack: vi.fn(),
    onSkip: vi.fn(),
    isFirst: false,
    isLast: false,
    ...overrides,
  };
}

describe('ApiKeysStep', () => {
  it('renders heading and description', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByText('API 金鑰')).toBeInTheDocument();
    expect(screen.getByText(/設定 API 金鑰以啟用進階功能/)).toBeInTheDocument();
  });

  it('renders TMDb key input', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByTestId('tmdb-key-input')).toBeInTheDocument();
  });

  it('renders AI provider select', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByTestId('ai-provider-select')).toBeInTheDocument();
  });

  it('does not show AI key input when no provider selected', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.queryByTestId('ai-key-input')).not.toBeInTheDocument();
  });

  it('shows AI key input when AI provider is selected', () => {
    render(<ApiKeysStep {...makeProps({ data: { aiProvider: 'gemini' } })} />);
    expect(screen.getByTestId('ai-key-input')).toBeInTheDocument();
  });

  it('AI key input has type password', () => {
    render(<ApiKeysStep {...makeProps({ data: { aiProvider: 'claude' } })} />);
    const input = screen.getByTestId('ai-key-input') as HTMLInputElement;
    expect(input.type).toBe('password');
  });

  it('calls onUpdate when TMDb key changes', () => {
    const onUpdate = vi.fn();
    render(<ApiKeysStep {...makeProps({ onUpdate })} />);
    fireEvent.change(screen.getByTestId('tmdb-key-input'), { target: { value: 'mykey123' } });
    expect(onUpdate).toHaveBeenCalledWith({ tmdbApiKey: 'mykey123' });
  });

  it('calls onUpdate when AI provider changes', () => {
    const onUpdate = vi.fn();
    render(<ApiKeysStep {...makeProps({ onUpdate })} />);
    fireEvent.change(screen.getByTestId('ai-provider-select'), { target: { value: 'gemini' } });
    expect(onUpdate).toHaveBeenCalledWith({ aiProvider: 'gemini' });
  });

  it('shows skip warning when no keys entered', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByTestId('skip-warning')).toBeInTheDocument();
    expect(screen.getByText(/限制部分功能/)).toBeInTheDocument();
  });

  it('hides skip warning when TMDb key is entered', () => {
    render(<ApiKeysStep {...makeProps({ data: { tmdbApiKey: 'abc123' } })} />);
    expect(screen.queryByTestId('skip-warning')).not.toBeInTheDocument();
  });

  it('hides skip warning when AI provider is selected', () => {
    render(<ApiKeysStep {...makeProps({ data: { aiProvider: 'gemini' } })} />);
    expect(screen.queryByTestId('skip-warning')).not.toBeInTheDocument();
  });

  it('shows skip button', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByTestId('skip-button')).toBeInTheDocument();
  });

  it('lists all AI provider options', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByText('不使用 AI')).toBeInTheDocument();
    expect(screen.getByText('Google Gemini')).toBeInTheDocument();
    expect(screen.getByText('Anthropic Claude')).toBeInTheDocument();
  });
});
