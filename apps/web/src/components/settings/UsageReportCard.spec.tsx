import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { UsageReportCard, USAGE_REPORT_DOCS_URL } from './UsageReportCard';

// Real QueryClient + a fetch mock: the card's loading / error / retry states
// come from TanStack itself (a mocked hook would hide the refetch-resets-
// pending behaviour, see project memory).

const PAYLOAD =
  '{"type":"event","payload":{"website":"11111111-2222-3333-4444-555555555555","hostname":"vido","url":"/usage-report","name":"weekly_usage","id":"3f2a6c1e-8f0b-4d5e-9a7c-1b2c3d4e5f60","ip":"127.0.0.1","data":{"version":"0.1.2","subtitles_auto_7d":7,"subtitles_embedded_7d":4,"subtitles_online_7d":2,"subtitles_asr_7d":1}}}';

const fetchMock = vi.fn();

function ok(data: unknown) {
  return { ok: true, status: 200, json: async () => ({ success: true, data }) };
}
function fail(status = 500) {
  return {
    ok: false,
    status,
    json: async () => ({ success: false, error: { code: 'INTERNAL_ERROR', message: 'db gone' } }),
  };
}

function wire(available: boolean, enabled: boolean, sent: boolean) {
  return {
    available,
    enabled,
    last_sent_at: sent ? '2026-10-04T12:00:00Z' : null,
    last_payload: sent ? PAYLOAD : null,
  };
}

function renderCard() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <UsageReportCard />
    </QueryClientProvider>
  );
}

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('UsageReportCard (infra-optin-usage-report-b1)', () => {
  it('shows the title, the explanation and the docs link', async () => {
    fetchMock.mockResolvedValue(ok(wire(true, false, false)));
    renderCard();

    expect(await screen.findByRole('switch', { name: '每週送一次匿名計數' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '匿名使用回報' })).toBeInTheDocument();
    expect(screen.getByText('每週最多一次，把幾個匿名數字送給 Vido 維護者。')).toBeInTheDocument();
    const link = screen.getByRole('link', { name: '送什麼、不送什麼 →' });
    expect(link).toHaveAttribute('href', USAGE_REPORT_DOCS_URL);
    expect(link).toHaveAttribute('target', '_blank');
  });

  it('unavailable: switch disabled, reason shown, no readout', async () => {
    fetchMock.mockResolvedValue(ok(wire(false, false, false)));
    renderCard();

    const sw = await screen.findByRole('switch');
    expect(sw).toBeDisabled();
    expect(sw).toHaveAttribute('aria-checked', 'false');
    expect(screen.getByText('這個版本沒有設定接收端，無法開啟。')).toBeInTheDocument();
    expect(screen.queryByText('上次送出')).not.toBeInTheDocument();
    expect(screen.queryByTestId('usage-report-payload')).not.toBeInTheDocument();
  });

  it('off (default): switch off and enabled, no readout', async () => {
    fetchMock.mockResolvedValue(ok(wire(true, false, false)));
    renderCard();

    const sw = await screen.findByRole('switch');
    expect(sw).toBeEnabled();
    expect(sw).toHaveAttribute('aria-checked', 'false');
    expect(screen.getByText('預設關閉。關掉之後就不會再送。')).toBeInTheDocument();
    expect(screen.queryByText('上次送出')).not.toBeInTheDocument();
  });

  it('on, never sent: readout says when the first one goes', async () => {
    fetchMock.mockResolvedValue(ok(wire(true, true, false)));
    renderCard();

    expect(await screen.findByRole('switch')).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByText('上次送出')).toBeInTheDocument();
    expect(screen.getByText('還沒送過——打開後一小時內會送出第一份')).toBeInTheDocument();
    expect(screen.queryByTestId('usage-report-payload')).not.toBeInTheDocument();
  });

  it('sent: shows the exact payload, byte for byte, labelled', async () => {
    fetchMock.mockResolvedValue(ok(wire(true, true, true)));
    renderCard();

    const payload = await screen.findByTestId('usage-report-payload');
    expect(payload.textContent).toBe(PAYLOAD);
    expect(payload).toHaveAccessibleName('送出的內容（原文）');
    expect(screen.getByText('上次送出')).toBeInTheDocument();
  });

  it('turned off after sending: the record of the last send stays visible', async () => {
    fetchMock.mockResolvedValue(ok(wire(true, false, true)));
    renderCard();

    expect(await screen.findByTestId('usage-report-payload')).toHaveTextContent('weekly_usage');
    expect(screen.getByRole('switch')).toHaveAttribute('aria-checked', 'false');
  });

  it('toggling saves through PUT and shows the server answer', async () => {
    fetchMock
      .mockResolvedValueOnce(ok(wire(true, false, false)))
      .mockResolvedValueOnce(ok(wire(true, true, false)));
    renderCard();
    const user = userEvent.setup();

    await user.click(await screen.findByRole('switch'));

    await waitFor(() => expect(screen.getByRole('switch')).toHaveAttribute('aria-checked', 'true'));
    const [url, init] = fetchMock.mock.calls[1];
    expect(url).toBe('/api/v1/settings/usage-report');
    expect(JSON.parse(init.body)).toEqual({ enabled: true });
    expect(screen.getByText('還沒送過——打開後一小時內會送出第一份')).toBeInTheDocument();
  });

  it('a failed save goes back and says so in plain words — never the backend text', async () => {
    fetchMock.mockResolvedValueOnce(ok(wire(true, false, false))).mockResolvedValueOnce(fail(500));
    renderCard();
    const user = userEvent.setup();

    await user.click(await screen.findByRole('switch'));

    expect(await screen.findByRole('alert')).toHaveTextContent('沒有存到，請再試一次。');
    expect(screen.getByRole('switch')).toHaveAttribute('aria-checked', 'false');
    expect(screen.queryByText(/db gone/)).not.toBeInTheDocument();
  });

  it('a failed load shows one line and a retry that works', async () => {
    fetchMock.mockResolvedValueOnce(fail(500)).mockResolvedValueOnce(ok(wire(true, false, false)));
    renderCard();
    const user = userEvent.setup();

    expect(await screen.findByText('讀不到匿名使用回報的狀態。')).toBeInTheDocument();
    expect(screen.queryByText(/db gone/)).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '重試' }));

    expect(await screen.findByRole('switch')).toBeInTheDocument();
  });
});
