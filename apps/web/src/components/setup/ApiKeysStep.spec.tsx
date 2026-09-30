import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { KeySettings } from '../../services/keySettingsService';

// sub-7-7b: the step asks /settings/keys whether a bundled TMDb key exists.
// Stubbed at the hook so these stay component tests (no QueryClient); the
// default is "nothing known" = the source-build copy every older test expects.
const h = vi.hoisted(() => ({
  query: { data: undefined as KeySettings | undefined },
}));

vi.mock('../../hooks/useKeySettings', () => ({
  useKeySettings: () => ({ ...h.query, isLoading: false, isError: false }),
}));

import { ApiKeysStep } from './ApiKeysStep';
import type { StepProps } from './SetupWizard';

const BUNDLED: KeySettings = {
  writable: true,
  keys: [
    { name: 'claude', configured: false, source: 'none' },
    { name: 'tmdb', configured: true, source: 'bundled' },
    { name: 'openai', configured: false, source: 'none' },
  ],
};

beforeEach(() => {
  h.query = { data: undefined };
});

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

describe('ApiKeysStep — bundled TMDb key (sub-7-7b)', () => {
  it('on a release image the TMDb field says the key is optional and skipping keeps metadata', () => {
    h.query = { data: BUNDLED };
    render(<ApiKeysStep {...makeProps()} />);

    expect(
      screen.getByText('已內建預設金鑰，可留空；填入自己的金鑰會優先使用')
    ).toBeInTheDocument();
    expect(screen.getByTestId('tmdb-key-input')).toHaveAttribute(
      'placeholder',
      '可留空，使用內建金鑰'
    );
    const warning = screen.getByTestId('skip-warning');
    expect(warning).toHaveTextContent('電影和影集的元資料會使用內建的 TMDb 金鑰');
    expect(warning).not.toHaveTextContent('自動取得元資料');
  });

  it('on a source build (no bundled key) the older, honest copy stays', () => {
    h.query = {
      data: {
        writable: true,
        keys: [
          { name: 'claude', configured: false, source: 'none' },
          { name: 'tmdb', configured: false, source: 'none' },
          { name: 'openai', configured: false, source: 'none' },
        ],
      },
    };
    render(<ApiKeysStep {...makeProps()} />);

    expect(screen.getByText('用於取得電影和影集的中文元資料')).toBeInTheDocument();
    expect(screen.getByTestId('skip-warning')).toHaveTextContent(
      '例如自動取得元資料和 AI 檔名解析'
    );
  });

  it('while the key state is unknown (request failed / still loading) it does not promise a bundled key', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.queryByText(/已內建預設金鑰/)).toBeNull();
  });
});
