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
    // setup-wizard-byok: say who pays before asking for a key.
    expect(
      screen.getByText(/費用直接記在你的 Claude 或 Gemini 帳號，不經過\s*Vido/)
    ).toBeInTheDocument();
    expect(screen.getByText(/Vido 不收錢/)).toBeInTheDocument();
  });

  it('renders the TMDb key input under the N4-D label', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByLabelText('TMDb 金鑰')).toBe(screen.getByTestId('tmdb-key-input'));
  });

  it('collects a Claude key and a Gemini key, each in its own field (setup-wizard-byok)', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByLabelText('Claude 金鑰')).toBe(screen.getByTestId('claude-key-input'));
    expect(screen.getByLabelText('Gemini 金鑰（選填）')).toBe(
      screen.getByTestId('gemini-key-input')
    );
  });

  it('offers no AI provider picker — each key has its own field (dsr-13)', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
  });

  it('masks both AI keys', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect((screen.getByTestId('claude-key-input') as HTMLInputElement).type).toBe('password');
    expect((screen.getByTestId('gemini-key-input') as HTMLInputElement).type).toBe('password');
  });

  it('links each AI key to where the provider hands it out, in a new tab', () => {
    render(<ApiKeysStep {...makeProps()} />);
    const claude = screen.getByTestId('claude-key-link');
    expect(claude).toHaveTextContent('怎麼申請 Claude 金鑰 ↗');
    // The ↗ is decoration; screen readers hear that it opens a new tab instead.
    expect(claude).toHaveAccessibleName(/^怎麼申請 Claude 金鑰\s*（開新分頁）$/);
    expect(claude).toHaveAttribute('href', 'https://console.anthropic.com/settings/keys');
    expect(claude).toHaveAttribute('target', '_blank');
    const gemini = screen.getByTestId('gemini-key-link');
    expect(gemini).toHaveTextContent('怎麼申請 Gemini 金鑰 ↗');
    expect(gemini).toHaveAccessibleName(/^怎麼申請 Gemini 金鑰\s*（開新分頁）$/);
    expect(gemini).toHaveAttribute('href', 'https://aistudio.google.com/apikey');
    expect(gemini).toHaveAttribute('rel', 'noreferrer');
  });

  it('describes each key with a hint the input points at', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByTestId('tmdb-key-input')).toHaveAccessibleDescription(
      '用於取得電影和影集的中文元資料'
    );
    expect(screen.getByTestId('claude-key-input')).toHaveAccessibleDescription(
      '預設的翻譯模型。一集影集（約一小時、有英文字幕）大約 US$0.15～0.3。'
    );
    // Never 「免費」 on its own — it is the user's own free quota.
    expect(screen.getByTestId('gemini-key-input')).toHaveAccessibleDescription(
      '沒有 Claude 金鑰？用你自己的 Gemini 免費額度也能翻；免費方案有速率限制，Vido 會自動放慢。'
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

  it('calls onUpdate with geminiApiKey when the Gemini key changes', () => {
    const onUpdate = vi.fn();
    render(<ApiKeysStep {...makeProps({ onUpdate })} />);
    fireEvent.change(screen.getByTestId('gemini-key-input'), { target: { value: 'AIza-1' } });
    expect(onUpdate).toHaveBeenCalledWith({ geminiApiKey: 'AIza-1' });
  });

  it('shows skip warning when no keys entered', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.getByTestId('skip-warning')).toBeInTheDocument();
    expect(screen.getByTestId('skip-warning')).toHaveTextContent(
      '沒填金鑰的話，就不能做 AI 字幕翻譯，也不能自動取得電影和影集的元資料。'
    );
  });

  it('hides skip warning when TMDb key is entered', () => {
    render(<ApiKeysStep {...makeProps({ data: { tmdbApiKey: 'abc123' } })} />);
    expect(screen.queryByTestId('skip-warning')).not.toBeInTheDocument();
  });

  it('hides skip warning when a Claude key is entered', () => {
    render(<ApiKeysStep {...makeProps({ data: { claudeApiKey: 'sk-ant-1' } })} />);
    expect(screen.queryByTestId('skip-warning')).not.toBeInTheDocument();
  });

  it('hides skip warning when only a Gemini key is entered', () => {
    render(<ApiKeysStep {...makeProps({ data: { geminiApiKey: 'AIza-1' } })} />);
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
    expect(warning).toHaveTextContent(
      'Claude 和 Gemini 金鑰都沒填的話，就不能做 AI 字幕翻譯；電影和影集的元資料會使用內建的 TMDb 金鑰。'
    );
    // AI filename parsing reads env keys only — a wizard key never enables it, so
    // the warning must not imply it does (setup-wizard-byok CR M2).
    expect(warning).not.toHaveTextContent('檔名解析');
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
      '也不能自動取得電影和影集的元資料'
    );
  });

  it('while the key state is unknown (request failed / still loading) it does not promise a bundled key', () => {
    render(<ApiKeysStep {...makeProps()} />);
    expect(screen.queryByText(/已內建預設金鑰/)).toBeNull();
  });
});
