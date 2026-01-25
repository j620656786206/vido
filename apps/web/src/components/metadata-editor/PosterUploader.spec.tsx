/**
 * PosterUploader Tests (Story 3.8 - AC3)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PosterUploader } from './PosterUploader';

describe('PosterUploader', () => {
  const defaultProps = {
    mediaId: 'test-media-id',
    onUpload: vi.fn().mockResolvedValue(undefined),
  };

  it('renders file upload tab by default', () => {
    render(<PosterUploader {...defaultProps} />);

    expect(screen.getByTestId('poster-dropzone')).toBeTruthy();
    expect(screen.getByText('拖放圖片或點擊選擇檔案')).toBeTruthy();
  });

  it('switches to URL input tab', async () => {
    render(<PosterUploader {...defaultProps} onUrlSubmit={vi.fn()} />);

    await userEvent.click(screen.getByRole('button', { name: /輸入網址/ }));

    expect(screen.getByTestId('poster-url-input')).toBeTruthy();
  });

  it('renders current poster as preview', () => {
    render(<PosterUploader {...defaultProps} currentPoster="https://example.com/poster.jpg" />);

    const preview = screen.getByAltText('Poster preview');
    expect(preview).toBeTruthy();
    expect((preview as HTMLImageElement).src).toBe('https://example.com/poster.jpg');
  });

  it('shows loading state during upload', () => {
    render(<PosterUploader {...defaultProps} isUploading />);

    expect(screen.getByText('上傳中...')).toBeTruthy();
  });

  it('shows error message', () => {
    render(<PosterUploader {...defaultProps} error="上傳失敗" />);

    expect(screen.getByText('上傳失敗')).toBeTruthy();
  });

  it('handles file selection', async () => {
    const onUpload = vi.fn().mockResolvedValue(undefined);
    render(<PosterUploader {...defaultProps} onUpload={onUpload} />);

    const file = new File(['dummy'], 'test.jpg', { type: 'image/jpeg' });
    const input = screen.getByTestId('poster-file-input');

    Object.defineProperty(input, 'files', {
      value: [file],
    });

    fireEvent.change(input);

    expect(onUpload).toHaveBeenCalledWith(file);
  });

  it('validates file type', async () => {
    const onUpload = vi.fn().mockResolvedValue(undefined);
    render(<PosterUploader {...defaultProps} onUpload={onUpload} />);

    const file = new File(['dummy'], 'test.gif', { type: 'image/gif' });
    const input = screen.getByTestId('poster-file-input');

    Object.defineProperty(input, 'files', {
      value: [file],
    });

    fireEvent.change(input);

    expect(onUpload).not.toHaveBeenCalled();
    expect(screen.getByText(/不支援的檔案格式/)).toBeTruthy();
  });

  it('validates file size', async () => {
    const onUpload = vi.fn().mockResolvedValue(undefined);
    render(<PosterUploader {...defaultProps} onUpload={onUpload} />);

    // Create a file larger than 5MB
    const largeContent = new Array(6 * 1024 * 1024).fill('a').join('');
    const file = new File([largeContent], 'large.jpg', { type: 'image/jpeg' });
    const input = screen.getByTestId('poster-file-input');

    Object.defineProperty(input, 'files', {
      value: [file],
    });

    fireEvent.change(input);

    expect(onUpload).not.toHaveBeenCalled();
    expect(screen.getByText(/檔案大小超過 5MB 限制/)).toBeTruthy();
  });

  it('handles URL submission', async () => {
    const onUrlSubmit = vi.fn();
    render(<PosterUploader {...defaultProps} onUrlSubmit={onUrlSubmit} />);

    await userEvent.click(screen.getByRole('button', { name: /輸入網址/ }));

    const urlInput = screen.getByTestId('poster-url-input');
    await userEvent.type(urlInput, 'https://example.com/poster.jpg');

    await userEvent.click(screen.getByRole('button', { name: '套用' }));

    expect(onUrlSubmit).toHaveBeenCalledWith('https://example.com/poster.jpg');
  });

  it('disables apply button when URL is empty', async () => {
    render(<PosterUploader {...defaultProps} onUrlSubmit={vi.fn()} />);

    await userEvent.click(screen.getByRole('button', { name: /輸入網址/ }));

    const applyButton = screen.getByRole('button', { name: '套用' });
    expect(applyButton.hasAttribute('disabled')).toBe(true);
  });

  it('clears preview on X click', async () => {
    render(<PosterUploader {...defaultProps} currentPoster="https://example.com/poster.jpg" />);

    await userEvent.click(screen.getByLabelText('清除預覽'));

    expect(screen.queryByAltText('Poster preview')).toBeNull();
    expect(screen.getByText('拖放圖片或點擊選擇檔案')).toBeTruthy();
  });

  it('shows file size limit info', () => {
    render(<PosterUploader {...defaultProps} />);

    expect(screen.getByText(/最大 5MB/)).toBeTruthy();
  });

  it('handles drag over state', () => {
    render(<PosterUploader {...defaultProps} />);

    const dropzone = screen.getByTestId('poster-dropzone');

    fireEvent.dragOver(dropzone);
    expect(dropzone.className).toContain('border-blue-500');

    fireEvent.dragLeave(dropzone);
    expect(dropzone.className).not.toContain('border-blue-500');
  });
});
