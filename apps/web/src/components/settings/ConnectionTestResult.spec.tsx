import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ConnectionTestResult } from './ConnectionTestResult';

describe('ConnectionTestResult', () => {
  it('renders success state with version info', () => {
    render(
      <ConnectionTestResult
        success={true}
        message="連線成功！"
        version="v4.5.2"
        apiVersion="2.9.3"
      />
    );

    expect(screen.getByTestId('connection-test-result')).toBeTruthy();
    expect(screen.getByText('連線成功！')).toBeTruthy();
    expect(screen.getByText('qBittorrent v4.5.2 (API 2.9.3)')).toBeTruthy();
  });

  it('renders success state without API version', () => {
    render(<ConnectionTestResult success={true} message="連線成功！" version="v4.5.2" />);

    expect(screen.getByText('qBittorrent v4.5.2')).toBeTruthy();
  });

  it('renders failure state', () => {
    render(<ConnectionTestResult success={false} message="連線失敗" />);

    expect(screen.getByText('連線失敗')).toBeTruthy();
    expect(screen.queryByText(/qBittorrent/)).toBeNull();
  });

  it('applies success styling', () => {
    render(<ConnectionTestResult success={true} message="Connected" version="v4.5.2" />);

    const el = screen.getByTestId('connection-test-result');
    expect(el.className).toContain('border-green-700');
  });

  it('applies failure styling', () => {
    render(<ConnectionTestResult success={false} message="Failed" />);

    const el = screen.getByTestId('connection-test-result');
    expect(el.className).toContain('border-red-700');
  });
});
