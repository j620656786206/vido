import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QBittorrentStep } from './QBittorrentStep';
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

describe('QBittorrentStep', () => {
  it('renders heading and description', () => {
    render(<QBittorrentStep {...makeProps()} />);
    expect(screen.getByText('qBittorrent 連線')).toBeInTheDocument();
    expect(screen.getByText(/連接 qBittorrent/)).toBeInTheDocument();
  });

  it('renders URL, username, and password inputs', () => {
    render(<QBittorrentStep {...makeProps()} />);
    expect(screen.getByTestId('qbt-url-input')).toBeInTheDocument();
    expect(screen.getByTestId('qbt-username-input')).toBeInTheDocument();
    expect(screen.getByTestId('qbt-password-input')).toBeInTheDocument();
  });

  it('password input has type password', () => {
    render(<QBittorrentStep {...makeProps()} />);
    const pwInput = screen.getByTestId('qbt-password-input') as HTMLInputElement;
    expect(pwInput.type).toBe('password');
  });

  it('calls onUpdate when URL changes', () => {
    const onUpdate = vi.fn();
    render(<QBittorrentStep {...makeProps({ onUpdate })} />);
    fireEvent.change(screen.getByTestId('qbt-url-input'), {
      target: { value: 'http://192.168.1.1:8080' },
    });
    expect(onUpdate).toHaveBeenCalledWith({ qbtUrl: 'http://192.168.1.1:8080' });
  });

  it('calls onUpdate when username changes', () => {
    const onUpdate = vi.fn();
    render(<QBittorrentStep {...makeProps({ onUpdate })} />);
    fireEvent.change(screen.getByTestId('qbt-username-input'), { target: { value: 'admin' } });
    expect(onUpdate).toHaveBeenCalledWith({ qbtUsername: 'admin' });
  });

  it('shows skip button when onSkip provided', () => {
    render(<QBittorrentStep {...makeProps()} />);
    expect(screen.getByTestId('skip-button')).toBeInTheDocument();
  });

  it('does not show skip button when onSkip not provided', () => {
    render(<QBittorrentStep {...makeProps({ onSkip: undefined })} />);
    expect(screen.queryByTestId('skip-button')).not.toBeInTheDocument();
  });

  it('calls onSkip when skip button clicked', () => {
    const onSkip = vi.fn();
    render(<QBittorrentStep {...makeProps({ onSkip })} />);
    fireEvent.click(screen.getByTestId('skip-button'));
    expect(onSkip).toHaveBeenCalledTimes(1);
  });

  it('calls onBack when back button clicked', () => {
    const onBack = vi.fn();
    render(<QBittorrentStep {...makeProps({ onBack })} />);
    fireEvent.click(screen.getByTestId('back-button'));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it('displays existing data from props', () => {
    render(
      <QBittorrentStep
        {...makeProps({ data: { qbtUrl: 'http://test:8080', qbtUsername: 'user1' } })}
      />
    );
    expect((screen.getByTestId('qbt-url-input') as HTMLInputElement).value).toBe(
      'http://test:8080'
    );
    expect((screen.getByTestId('qbt-username-input') as HTMLInputElement).value).toBe('user1');
  });
});
