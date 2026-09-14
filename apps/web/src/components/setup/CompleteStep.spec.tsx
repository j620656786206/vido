import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { CompleteStep } from './CompleteStep';
import type { StepProps } from './SetupWizard';

function makeProps(overrides?: Partial<StepProps>): StepProps {
  return {
    data: { language: 'zh-TW', libraries: [{ path: '/media', contentType: 'movie' }] },
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

  it('shows the language by name, not by code', () => {
    render(<CompleteStep {...makeProps()} />);
    expect(screen.getByText('繁體中文')).toBeInTheDocument();
    expect(screen.queryByText('zh-TW')).not.toBeInTheDocument();
  });

  // Before dsr-13 this row read the deprecated mediaFolderPath, which the wizard
  // stopped setting when it moved to libraries — so it said 未設定 on every real run.
  it('summarises the libraries the wizard collected: count and kinds', () => {
    render(
      <CompleteStep
        {...makeProps({
          data: {
            language: 'zh-TW',
            libraries: [
              { path: '/media/movies', contentType: 'movie' },
              { path: '/media/tv', contentType: 'series' },
            ],
          },
        })}
      />
    );
    expect(screen.getByText('2 個（電影・影集）')).toBeInTheDocument();
  });

  it('names each kind once', () => {
    render(
      <CompleteStep
        {...makeProps({
          data: {
            language: 'zh-TW',
            libraries: [
              { path: '/a', contentType: 'movie' },
              { path: '/b', contentType: 'movie' },
            ],
          },
        })}
      />
    );
    expect(screen.getByText('2 個（電影）')).toBeInTheDocument();
  });

  it('does not count a library row whose path is blank', () => {
    render(
      <CompleteStep
        {...makeProps({
          data: {
            language: 'zh-TW',
            libraries: [
              { path: '/a', contentType: 'movie' },
              { path: '   ', contentType: 'series' },
            ],
          },
        })}
      />
    );
    expect(screen.getByText('1 個（電影）')).toBeInTheDocument();
  });

  it('shows "未設定" for every missing optional field', () => {
    render(<CompleteStep {...makeProps({ data: { language: 'en' } })} />);
    // 媒體資料夾 · qBittorrent · TMDb 金鑰 · Claude 金鑰
    expect(screen.getAllByText('未設定')).toHaveLength(4);
  });

  it('says 已設定 for qBittorrent instead of echoing the URL', () => {
    render(
      <CompleteStep {...makeProps({ data: { language: 'en', qbtUrl: 'http://qbt:8080' } })} />
    );
    expect(screen.getByText('已設定')).toBeInTheDocument();
    expect(screen.queryByText('http://qbt:8080')).not.toBeInTheDocument();
  });

  it('shows "已設定" for TMDb when key exists', () => {
    render(<CompleteStep {...makeProps({ data: { language: 'en', tmdbApiKey: 'key123' } })} />);
    expect(screen.getByText('已設定')).toBeInTheDocument();
  });

  it('shows "已設定" for the Claude key when one was entered', () => {
    render(<CompleteStep {...makeProps({ data: { language: 'en', claudeApiKey: 'sk-ant-1' } })} />);
    expect(screen.getByText('Claude 金鑰')).toBeInTheDocument();
    expect(screen.getByText('已設定')).toBeInTheDocument();
  });

  it('tells the user unset items can be filled in later', () => {
    render(<CompleteStep {...makeProps()} />);
    expect(screen.getByText('未設定的項目之後都可以在「設定」裡補上。')).toBeInTheDocument();
  });

  it('has finish button (not next)', () => {
    render(<CompleteStep {...makeProps()} />);
    expect(screen.getByTestId('finish-button')).toBeInTheDocument();
    expect(screen.getByTestId('finish-button')).toHaveTextContent('完成設定');
  });

  it('finishes with the primary action colour, not the success colour', () => {
    render(<CompleteStep {...makeProps()} />);
    const finish = screen.getByTestId('finish-button');
    expect(finish.className).toContain('bg-[var(--accent-primary)]');
    expect(finish.className).not.toMatch(/--success/);
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
