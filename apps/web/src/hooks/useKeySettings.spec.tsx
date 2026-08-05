import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useKeySettings, useSaveKeys, keySettingsQueryKeys } from './useKeySettings';
import { keySettingsService, type KeySettings } from '../services/keySettingsService';

vi.mock('../services/keySettingsService', () => ({
  keySettingsService: { getKeys: vi.fn(), saveKeys: vi.fn(), testClaudeKey: vi.fn() },
}));

const mockedGet = vi.mocked(keySettingsService.getKeys);
const mockedSave = vi.mocked(keySettingsService.saveKeys);

const INITIAL: KeySettings = {
  writable: true,
  keys: [{ name: 'claude', configured: false, source: 'none' }],
};

/** The state a PUT returns — claude flipped to secret-sourced. */
const AFTER_SAVE: KeySettings = {
  writable: true,
  keys: [{ name: 'claude', configured: true, source: 'secret', masked: 'sk-ant…7f3a' }],
};

function createHarness() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { queryClient, wrapper };
}

beforeEach(() => {
  vi.clearAllMocks();
  mockedGet.mockResolvedValue(INITIAL);
  mockedSave.mockResolvedValue(AFTER_SAVE);
});

describe('keySettingsQueryKeys', () => {
  it('list key extends the all key', () => {
    expect(keySettingsQueryKeys.all).toEqual(['settings', 'keys']);
    expect(keySettingsQueryKeys.list()).toEqual(['settings', 'keys', 'list']);
  });
});

describe('useKeySettings', () => {
  it('fetches the key state through the service', async () => {
    const { wrapper } = createHarness();
    const { result } = renderHook(() => useKeySettings(), { wrapper });

    await waitFor(() => expect(result.current.data).toEqual(INITIAL));
    expect(mockedGet).toHaveBeenCalledTimes(1);
  });
});

describe('useSaveKeys', () => {
  // CR sub-2-1b M4: the PUT response IS the fresh state and must seed the cache
  // directly — the env→secret flip is the user-visible proof the save took
  // effect, and it must not wait on a second round trip.
  it('seeds the query cache with the PUT response instead of refetching', async () => {
    const { queryClient, wrapper } = createHarness();
    const query = renderHook(() => useKeySettings(), { wrapper });
    await waitFor(() => expect(query.result.current.data).toEqual(INITIAL));

    const save = renderHook(() => useSaveKeys(), { wrapper });
    act(() => save.result.current.mutate({ claude: 'sk-ant-new' }));

    await waitFor(() =>
      expect(queryClient.getQueryData(keySettingsQueryKeys.list())).toEqual(AFTER_SAVE)
    );
    // The reader sees the new state without a second GET.
    await waitFor(() => expect(query.result.current.data).toEqual(AFTER_SAVE));
    expect(mockedGet).toHaveBeenCalledTimes(1);
  });

  it('leaves the cache untouched when the save is refused', async () => {
    const { queryClient, wrapper } = createHarness();
    const query = renderHook(() => useKeySettings(), { wrapper });
    await waitFor(() => expect(query.result.current.data).toEqual(INITIAL));

    mockedSave.mockRejectedValue(new Error('未設定加密金鑰，無法安全儲存 API 金鑰'));
    const save = renderHook(() => useSaveKeys(), { wrapper });
    act(() => save.result.current.mutate({ claude: 'sk-ant-new' }));

    await waitFor(() => expect(save.result.current.isError).toBe(true));
    expect(queryClient.getQueryData(keySettingsQueryKeys.list())).toEqual(INITIAL);
  });
});
