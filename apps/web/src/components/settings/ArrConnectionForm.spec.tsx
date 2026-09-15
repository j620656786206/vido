import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  DVR_AUTH_FAILED,
  DVR_CONNECTION_FAILED,
  DVR_TEST_FAILED,
  DvrSettingsApiError,
  type DvrConfig,
} from '../../services/dvrSettings';

type MutateOpts = { onSuccess?: () => void; onError?: (e: Error) => void };
type ListState<T> = { data: T[] | undefined; isLoading: boolean; isError: boolean };

const h = vi.hoisted(() => ({
  config: undefined as unknown,
  configState: { isLoading: false, isError: false, error: null as Error | null },
  saveMutate: vi.fn(),
  saveState: { isPending: false, isError: false, error: null as Error | null },
  testMutate: vi.fn(),
  profiles: { data: [], isLoading: false, isError: false } as ListState<{
    id: number;
    name: string;
  }>,
  folders: { data: [], isLoading: false, isError: false } as ListState<{
    id: number;
    path: string;
  }>,
  profilesEnabled: [] as boolean[],
  foldersEnabled: [] as boolean[],
}));

vi.mock('../../hooks/useDvrSettings', () => ({
  useDvrConfig: () => ({ data: h.config, ...h.configState, refetch: vi.fn(), isFetching: false }),
  useSaveDvrConfig: () => ({ mutate: h.saveMutate, ...h.saveState }),
  useTestDvrConnection: () => ({ mutate: h.testMutate, isPending: false }),
  // The real hooks only fetch when `enabled`; mirror that so the gate is actually tested.
  useDvrQualityProfiles: (_plugin: string, enabled: boolean) => {
    h.profilesEnabled.push(enabled);
    return enabled ? h.profiles : { data: undefined, isLoading: false, isError: false };
  },
  useDvrRootFolders: (_plugin: string, enabled: boolean) => {
    h.foldersEnabled.push(enabled);
    return enabled ? h.folders : { data: undefined, isLoading: false, isError: false };
  },
}));

import { ArrConnectionForm, describeDvrError } from './ArrConnectionForm';

const NEVER_SET_UP: DvrConfig = {
  url: '',
  enabled: false,
  qualityProfileId: 0,
  rootFolderPath: '',
  hasApiKey: false,
  health: { status: 'unconfigured', lastCheckedAt: null, message: 'plugin not configured' },
};

const saved = (over: Partial<DvrConfig> = {}): DvrConfig => ({
  url: 'http://nas:8989',
  enabled: true,
  qualityProfileId: 4,
  rootFolderPath: '/data/media/tv',
  hasApiKey: true,
  health: { status: 'healthy', lastCheckedAt: '2026-09-15T00:00:00Z', message: '' },
  ...over,
});

beforeEach(() => {
  vi.clearAllMocks();
  h.config = NEVER_SET_UP;
  h.configState = { isLoading: false, isError: false, error: null };
  h.saveState = { isPending: false, isError: false, error: null };
  h.profiles = {
    data: [
      { id: 4, name: 'HD-1080p' },
      { id: 6, name: 'Ultra-HD' },
    ],
    isLoading: false,
    isError: false,
  };
  h.folders = { data: [{ id: 1, path: '/data/media/tv' }], isLoading: false, isError: false };
  h.profilesEnabled = [];
  h.foldersEnabled = [];
});

describe('ArrConnectionForm — never set up (13-6)', () => {
  it('says 未設定, opens switched on, asks for the key, and keeps both lists locked', async () => {
    const user = userEvent.setup();
    render(<ArrConnectionForm plugin="sonarr" />);

    expect(screen.getByRole('heading', { name: 'Sonarr' })).toBeInTheDocument();
    expect(screen.getByTestId('arr-health-sonarr')).toHaveTextContent('未設定');
    expect(screen.getByRole('switch', { name: '啟用' })).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByLabelText('API 金鑰')).toHaveAttribute(
      'placeholder',
      '貼上 Sonarr 的 API 金鑰'
    );
    expect(screen.getByLabelText('品質設定檔')).toBeDisabled();
    expect(screen.getByLabelText('品質設定檔')).toHaveDisplayValue('啟用並儲存連線後可以選擇');
    expect(screen.getByLabelText('根資料夾')).toBeDisabled();
    expect(h.profilesEnabled.every((e) => e === false)).toBe(true);
    expect(h.foldersEnabled.every((e) => e === false)).toBe(true);

    const testBtn = screen.getByRole('button', { name: '測試連線' });
    const saveBtn = screen.getByRole('button', { name: '儲存設定' });
    expect(testBtn).toBeDisabled();
    expect(saveBtn).toBeDisabled();

    await user.type(screen.getByLabelText('網址'), '192.168.1.100:8989');
    expect(saveBtn).toBeDisabled(); // no key yet
    await user.type(screen.getByLabelText('API 金鑰'), 'abc123');
    expect(testBtn).toBeEnabled();
    expect(saveBtn).toBeEnabled();
  });

  it('uses the Radarr name and port for the Radarr card', () => {
    render(<ArrConnectionForm plugin="radarr" />);
    expect(screen.getByRole('heading', { name: 'Radarr' })).toBeInTheDocument();
    expect(screen.getByLabelText('網址')).toHaveAttribute(
      'placeholder',
      'http://192.168.1.100:7878'
    );
  });
});

describe('ArrConnectionForm — saved connection', () => {
  it('shows 已連線, says the key is stored, and loads the lists with the saved choices', () => {
    h.config = saved();
    render(<ArrConnectionForm plugin="sonarr" />);

    expect(screen.getByTestId('arr-health-sonarr')).toHaveTextContent('已連線');
    expect(screen.getByText('已儲存。留空表示不變更。')).toBeInTheDocument();
    expect(h.profilesEnabled).toContain(true);
    expect(h.foldersEnabled).toContain(true);
    expect(screen.getByLabelText('品質設定檔')).toHaveValue('4');
    expect(screen.getByLabelText('根資料夾')).toHaveValue('/data/media/tv');
    expect(screen.queryByTestId('arr-setup-note')).toBeNull();
    // the stored key is enough to test or save
    expect(screen.getByRole('button', { name: '儲存設定' })).toBeEnabled();
  });

  it.each([
    [saved({ health: { status: 'unhealthy', lastCheckedAt: null, message: 'timeout' } }), '連不上'],
    [saved({ enabled: false }), '已停用'],
    [saved({ hasApiKey: false }), '未設定'],
    [
      saved({ health: { status: 'unconfigured', lastCheckedAt: null, message: 'not checked' } }),
      '尚未檢查',
    ],
  ])('the badge tells the truth: %#', (config, label) => {
    h.config = config;
    render(<ArrConnectionForm plugin="sonarr" />);
    expect(screen.getByTestId('arr-health-sonarr')).toHaveTextContent(label);
  });

  it('save sends every field, keeps the stored key, then clears the key box and says 設定已儲存', async () => {
    h.config = saved();
    h.saveMutate.mockImplementation((_params: unknown, opts: MutateOpts) => opts.onSuccess?.());
    const user = userEvent.setup();
    render(<ArrConnectionForm plugin="sonarr" />);

    await user.selectOptions(screen.getByLabelText('品質設定檔'), '6');
    await user.click(screen.getByRole('switch', { name: '啟用' }));
    await user.click(screen.getByRole('button', { name: '儲存設定' }));

    expect(h.saveMutate).toHaveBeenCalledWith(
      {
        url: 'http://nas:8989',
        apiKey: '',
        enabled: false,
        qualityProfileId: 6,
        rootFolderPath: '/data/media/tv',
      },
      expect.any(Object)
    );
    expect(screen.getByText('設定已儲存')).toBeInTheDocument();
    expect(screen.getByLabelText('API 金鑰')).toHaveValue('');

    // any further edit retracts the confirmation
    await user.selectOptions(screen.getByLabelText('品質設定檔'), '4');
    expect(screen.queryByText('設定已儲存')).toBeNull();
  });

  it('a new URL needs the key again — the stored key never goes to a different address', async () => {
    h.config = saved();
    const user = userEvent.setup();
    render(<ArrConnectionForm plugin="sonarr" />);

    const urlInput = screen.getByLabelText('網址');
    await user.clear(urlInput);
    await user.type(urlInput, 'http://other:8989');
    expect(screen.getByText('換了網址，要重新貼上 API 金鑰。')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '儲存設定' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '測試連線' })).toBeDisabled();

    await user.type(screen.getByLabelText('API 金鑰'), 'fresh');
    expect(screen.getByRole('button', { name: '儲存設定' })).toBeEnabled();
  });

  it('a saved, enabled connection without a profile or folder says requests will stall', () => {
    h.config = saved({ qualityProfileId: 0, rootFolderPath: '' });
    render(<ArrConnectionForm plugin="sonarr" />);
    expect(screen.getByTestId('arr-setup-note')).toHaveTextContent('請求會停在等待中');
  });

  it('a saved profile the server no longer has is shown as missing, not as 請選擇', () => {
    h.config = saved({ qualityProfileId: 7 });
    render(<ArrConnectionForm plugin="sonarr" />);
    expect(screen.getByLabelText('品質設定檔')).toHaveDisplayValue('找不到這個設定檔（#7）');
    expect(screen.getByTestId('arr-setup-note')).toHaveTextContent('找不到了');
  });

  it('a server with no root folders says where to add one', () => {
    h.config = saved({ rootFolderPath: '' });
    h.folders = { data: [], isLoading: false, isError: false };
    render(<ArrConnectionForm plugin="sonarr" />);
    expect(screen.getByText(/Sonarr 裡還沒有根資料夾/)).toBeInTheDocument();
  });

  it('a refused save names both facts: nothing was stored, and why (the wrapped cause)', async () => {
    h.config = saved();
    h.saveState = {
      isPending: false,
      isError: true,
      error: new DvrSettingsApiError(
        '儲存 Sonarr 設定失敗',
        DVR_TEST_FAILED,
        'sonarr rejected the API key',
        DVR_AUTH_FAILED
      ),
    };
    render(<ArrConnectionForm plugin="sonarr" />);
    const alert = screen.getByRole('alert');
    expect(alert).toHaveTextContent('設定沒有儲存');
    expect(alert).toHaveTextContent('API 金鑰不對');
    expect(alert).toHaveClass('text-[var(--warning-text)]');
  });
});

describe('ArrConnectionForm — test connection', () => {
  it('tests the typed URL and key, and an auth failure tells the user to recopy the key', async () => {
    h.testMutate.mockImplementation((_params: unknown, opts: MutateOpts) =>
      opts.onError?.(new DvrSettingsApiError('無法連線到 Sonarr', DVR_AUTH_FAILED))
    );
    const user = userEvent.setup();
    render(<ArrConnectionForm plugin="sonarr" />);

    await user.type(screen.getByLabelText('網址'), ' http://nas:8989 ');
    await user.type(screen.getByLabelText('API 金鑰'), 'wrong');
    await user.click(screen.getByRole('button', { name: '測試連線' }));

    expect(h.testMutate).toHaveBeenCalledWith(
      { url: 'http://nas:8989', apiKey: 'wrong' },
      expect.any(Object)
    );
    expect(screen.getByTestId('arr-test-result-sonarr')).toHaveTextContent('API 金鑰不對');
  });

  it('a Sonarr v3 refusal on test shows its own reason, not "press 測試連線"', async () => {
    h.config = saved();
    h.testMutate.mockImplementation((_params: unknown, opts: MutateOpts) =>
      opts.onError?.(
        new DvrSettingsApiError(
          '無法連線到 Sonarr',
          DVR_TEST_FAILED,
          '需要 Sonarr v4（偵測到 3.0.10）— v3 series adds require the removed languageProfileId',
          DVR_TEST_FAILED
        )
      )
    );
    const user = userEvent.setup();
    render(<ArrConnectionForm plugin="sonarr" />);
    await user.click(screen.getByRole('button', { name: '測試連線' }));
    const result = screen.getByTestId('arr-test-result-sonarr');
    expect(result).toHaveTextContent('需要 Sonarr v4（偵測到 3.0.10）');
    expect(result).not.toHaveTextContent('測試連線');
  });

  it('a successful test says so', async () => {
    h.config = saved();
    h.testMutate.mockImplementation((_params: unknown, opts: MutateOpts) => opts.onSuccess?.());
    const user = userEvent.setup();
    render(<ArrConnectionForm plugin="sonarr" />);
    await user.click(screen.getByRole('button', { name: '測試連線' }));
    expect(screen.getByTestId('arr-test-result-sonarr')).toHaveTextContent('連得到 Sonarr');
  });
});

describe('ArrConnectionForm — load failure', () => {
  it('replaces the form instead of showing an empty one that claims nothing is saved', () => {
    h.config = undefined;
    h.configState = { isLoading: false, isError: true, error: new Error('boom') };
    render(<ArrConnectionForm plugin="sonarr" />);
    expect(screen.queryByTestId('arr-form-sonarr')).toBeNull();
    expect(screen.getByText('無法讀取 Sonarr 設定')).toBeInTheDocument();
  });
});

describe('describeDvrError', () => {
  it('turns backend codes into something the user can act on', () => {
    expect(
      describeDvrError(new DvrSettingsApiError('x', DVR_CONNECTION_FAILED), 'Radarr', 'test')
    ).toContain('確認 Radarr 有在執行');
    expect(describeDvrError(new Error('plain message'), 'Radarr', 'test')).toBe('plain message');
  });

  it('a refused save whose cause is not a known code still says nothing was stored', () => {
    const msg = describeDvrError(
      new DvrSettingsApiError(
        '儲存 Radarr 設定失敗',
        DVR_TEST_FAILED,
        'radarr connection test failed — config not saved',
        DVR_TEST_FAILED
      ),
      'Radarr',
      'save'
    );
    expect(msg).toBe('連線測試沒有通過，設定沒有儲存。按「測試連線」看看原因。');
  });
});
