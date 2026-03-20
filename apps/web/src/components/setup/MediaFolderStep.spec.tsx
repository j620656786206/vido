import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { MediaFolderStep } from './MediaFolderStep';
import type { StepProps } from './SetupWizard';

function makeProps(overrides?: Partial<StepProps>): StepProps {
  return {
    data: {},
    onUpdate: vi.fn(),
    onNext: vi.fn(),
    onBack: vi.fn(),
    isFirst: false,
    isLast: false,
    ...overrides,
  };
}

describe('MediaFolderStep', () => {
  it('renders heading and description', () => {
    render(<MediaFolderStep {...makeProps()} />);
    expect(screen.getByText('媒體資料夾')).toBeInTheDocument();
    expect(screen.getByText(/媒體檔案存放路徑/)).toBeInTheDocument();
  });

  it('renders folder path input with placeholder', () => {
    render(<MediaFolderStep {...makeProps()} />);
    const input = screen.getByTestId('media-folder-input') as HTMLInputElement;
    expect(input).toBeInTheDocument();
    expect(input.placeholder).toBe('/media/videos');
  });

  it('calls onUpdate when path changes', () => {
    const onUpdate = vi.fn();
    render(<MediaFolderStep {...makeProps({ onUpdate })} />);
    fireEvent.change(screen.getByTestId('media-folder-input'), {
      target: { value: '/nas/movies' },
    });
    expect(onUpdate).toHaveBeenCalledWith({ mediaFolderPath: '/nas/movies' });
  });

  it('displays existing path from data prop', () => {
    render(<MediaFolderStep {...makeProps({ data: { mediaFolderPath: '/existing/path' } })} />);
    expect((screen.getByTestId('media-folder-input') as HTMLInputElement).value).toBe(
      '/existing/path'
    );
  });

  it('has accessible label for folder path input', () => {
    render(<MediaFolderStep {...makeProps()} />);
    expect(screen.getByLabelText('資料夾路徑')).toBeInTheDocument();
  });

  it('has back and next buttons', () => {
    render(<MediaFolderStep {...makeProps()} />);
    expect(screen.getByTestId('back-button')).toBeInTheDocument();
    expect(screen.getByTestId('next-button')).toBeInTheDocument();
  });

  it('does not have skip button (media folder is required)', () => {
    render(<MediaFolderStep {...makeProps()} />);
    expect(screen.queryByTestId('skip-button')).not.toBeInTheDocument();
  });
});
