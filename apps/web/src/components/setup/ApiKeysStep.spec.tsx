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

  it('renders the TMDb key input under the N4-D label', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByLabelText('TMDb 金鑰')).toBe(screen.getByTestId('tmdb-key-input'));
  });

  it('collects a Claude key — the only AI key the server can read back (dsr-13)', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByLabelText('Claude 金鑰')).toBe(screen.getByTestId('claude-key-input'));
  });

  it('offers no AI provider picker — a Gemini key typed here was stored where nothing read it', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    expect(screen.queryByText('Google Gemini')).not.toBeInTheDocument();
  });

  it('masks the Claude key', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect((screen.getByTestId('claude-key-input') as HTMLInputElement).type).toBe('password');
  });

  it('describes each key with a hint the input points at', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByTestId('tmdb-key-input')).toHaveAccessibleDescription(
      '用於取得電影和影集的中文元資料'
    );
    expect(screen.getByTestId('claude-key-input')).toHaveAccessibleDescription(
      '用於字幕翻譯與 AI 檔名解析'
    );
  });

  it('calls onUpdate when TMDb key changes', () => {
    const onUpdate = vi.fn();
    render(<ApiKeysStep {...makeProps({ onUpdate })} />);
    fireEvent.change(screen.getByTestId('tmdb-key-input'), { target: { value: 'mykey123' } });
    expect(onUpdate).toHaveBeenCalledWith({ tmdbApiKey: 'mykey123' });
  });

  it('calls onUpdate with claudeApiKey when the Claude key changes', () => {
    const onUpdate = vi.fn();
    render(<ApiKeysStep {...makeProps({ onUpdate })} />);
    fireEvent.change(screen.getByTestId('claude-key-input'), { target: { value: 'sk-ant-1' } });
    expect(onUpdate).toHaveBeenCalledWith({ claudeApiKey: 'sk-ant-1' });
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

  it('hides skip warning when a Claude key is entered', () => {
    render(<ApiKeysStep {...makeProps({ data: { claudeApiKey: 'sk-ant-1' } })} />);
    expect(screen.queryByTestId('skip-warning')).not.toBeInTheDocument();
  });

  it('keeps the skip warning out of the status palette (DESIGN.md 2026-09-11: 赭說的是現在的世界)', () => {
    render(<ApiKeysStep {...makeProps()} />);
    const warning = screen.getByTestId('skip-warning');
    expect(warning.className).toContain('bg-[var(--bg-tertiary)]');
    expect(warning.outerHTML).not.toMatch(/--warning/);
  });

  it('orders the buttons 上一步 · 跳過 · 下一步', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getAllByRole('button').map((b) => b.textContent)).toEqual([
      '上一步',
      '跳過',
      '下一步',
    ]);
  });
});
