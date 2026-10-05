import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { UsageReportStep } from './UsageReportStep';
import { USAGE_REPORT_DOCS_URL } from '../../services/usageReportService';
import type { StepProps } from './SetupWizard';

function renderStep(over: Partial<StepProps> = {}) {
  const props: StepProps = {
    data: {},
    onUpdate: vi.fn(),
    onNext: vi.fn(),
    onBack: vi.fn(),
    isFirst: false,
    isLast: false,
    ...over,
  };
  render(<UsageReportStep {...props} />);
  return props;
}

describe('UsageReportStep (infra-optin-usage-report-b2, N6-D)', () => {
  it('asks the question with the switch OFF by default', () => {
    renderStep();
    expect(screen.getByRole('heading', { name: '匿名使用回報' })).toBeInTheDocument();
    expect(
      screen.getByText('要不要每週告訴維護者「Vido 有在幫你做字幕」？只送幾個數字，預設關閉。')
    ).toBeInTheDocument();
    expect(screen.getByRole('switch', { name: '每週送一次匿名計數' })).toHaveAttribute(
      'aria-checked',
      'false'
    );
    expect(screen.getByText('之後隨時可以在「設定 → 連線設定」改。')).toBeInTheDocument();
  });

  it('lists what is sent and what never is, each in its own box', () => {
    renderStep();
    const facts = screen.getByTestId('usage-report-facts');
    const box = (title: string) => {
      const heading = within(facts).getByRole('heading', { name: title });
      return within(heading.parentElement as HTMLElement)
        .getAllByRole('listitem')
        .map((li) => li.textContent);
    };
    expect(box('會送')).toEqual([
      '一個隨機編號（不是從你的機器算出來的）',
      'Vido 版本',
      '最近 7 天 Vido 自己做出的字幕數',
    ]);
    expect(box('絕不送')).toEqual([
      '片名、檔名、資料夾路徑',
      'API 金鑰與任何設定',
      '你看了、想要或下載了什麼',
    ]);
  });

  it('toggling writes usageReportEnabled into the wizard data', () => {
    const props = renderStep();
    fireEvent.click(screen.getByRole('switch'));
    expect(props.onUpdate).toHaveBeenCalledWith({ usageReportEnabled: true });
  });

  it('shows ON when the wizard data says so, and toggles back off', () => {
    const props = renderStep({ data: { usageReportEnabled: true } });
    const sw = screen.getByRole('switch');
    expect(sw).toHaveAttribute('aria-checked', 'true');
    fireEvent.click(sw);
    expect(props.onUpdate).toHaveBeenCalledWith({ usageReportEnabled: false });
  });

  it('has only 上一步 and 下一步 — no 跳過', () => {
    const props = renderStep();
    expect(screen.queryByTestId('skip-button')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('back-button'));
    fireEvent.click(screen.getByTestId('next-button'));
    expect(props.onBack).toHaveBeenCalledTimes(1);
    expect(props.onNext).toHaveBeenCalledTimes(1);
  });

  it('links to the full explanation', () => {
    renderStep();
    expect(screen.getByRole('link', { name: '看完整說明 →' })).toHaveAttribute(
      'href',
      USAGE_REPORT_DOCS_URL
    );
  });
});
