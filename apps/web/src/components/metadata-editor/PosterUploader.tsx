/**
 * PosterUploader Component (Story 3.8 - AC3)
 * Drag-drop and file picker for poster upload
 */

import { useState, useCallback, useRef } from 'react';
import { Upload, Link, Loader2, X, Image } from 'lucide-react';
import { cn } from '../../lib/utils';

export interface PosterUploaderProps {
  mediaId: string;
  currentPoster?: string;
  onUpload: (file: File) => Promise<void>;
  onUrlSubmit?: (url: string) => void;
  isUploading?: boolean;
  error?: string | null;
}

type UploadMethod = 'file' | 'url';

const ACCEPTED_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
const MAX_SIZE = 5 * 1024 * 1024; // 5MB

export function PosterUploader({
  mediaId: _mediaId,
  currentPoster,
  onUpload,
  onUrlSubmit,
  isUploading = false,
  error,
}: PosterUploaderProps) {
  const [preview, setPreview] = useState<string | null>(currentPoster || null);
  const [uploadMethod, setUploadMethod] = useState<UploadMethod>('file');
  const [urlInput, setUrlInput] = useState('');
  const [isDragOver, setIsDragOver] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const validateFile = useCallback((file: File): string | null => {
    if (!ACCEPTED_TYPES.includes(file.type)) {
      return '不支援的檔案格式，請上傳 JPG、PNG 或 WebP 圖片';
    }
    if (file.size > MAX_SIZE) {
      return '檔案大小超過 5MB 限制';
    }
    return null;
  }, []);

  const handleFile = useCallback(
    async (file: File) => {
      const error = validateFile(file);
      if (error) {
        setValidationError(error);
        return;
      }

      setValidationError(null);
      setPreview(URL.createObjectURL(file));
      await onUpload(file);
    },
    [validateFile, onUpload]
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDragOver(false);

      const file = e.dataTransfer.files[0];
      if (file) {
        handleFile(file);
      }
    },
    [handleFile]
  );

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
  }, []);

  const handleFileSelect = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) {
        handleFile(file);
      }
    },
    [handleFile]
  );

  const handleUrlSubmit = useCallback(() => {
    if (urlInput.trim() && onUrlSubmit) {
      setPreview(urlInput.trim());
      onUrlSubmit(urlInput.trim());
      setUrlInput('');
    }
  }, [urlInput, onUrlSubmit]);

  const clearPreview = useCallback(() => {
    setPreview(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, []);

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-[var(--text-secondary)]">海報圖片</label>

      {/* Method Toggle */}
      <div className="flex rounded-lg bg-[var(--bg-secondary)] p-1 w-fit">
        <button
          type="button"
          onClick={() => setUploadMethod('file')}
          className={cn(
            'px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1.5',
            uploadMethod === 'file'
              ? 'bg-[var(--accent-primary)] text-white'
              : 'text-[var(--text-secondary)] hover:text-white'
          )}
        >
          <Upload className="h-4 w-4" />
          上傳檔案
        </button>
        <button
          type="button"
          onClick={() => setUploadMethod('url')}
          className={cn(
            'px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1.5',
            uploadMethod === 'url'
              ? 'bg-[var(--accent-primary)] text-white'
              : 'text-[var(--text-secondary)] hover:text-white'
          )}
        >
          <Link className="h-4 w-4" />
          輸入網址
        </button>
      </div>

      {uploadMethod === 'file' ? (
        <div
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onClick={() => fileInputRef.current?.click()}
          className={cn(
            'border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors',
            isDragOver
              ? 'border-[var(--accent-primary)] bg-[var(--accent-primary)]/10'
              : 'border-[var(--border-subtle)] hover:border-[var(--border-subtle)]',
            isUploading && 'opacity-50 pointer-events-none'
          )}
          data-testid="poster-dropzone"
        >
          <input
            ref={fileInputRef}
            type="file"
            accept=".jpg,.jpeg,.png,.webp"
            onChange={handleFileSelect}
            className="hidden"
            data-testid="poster-file-input"
          />
          {isUploading ? (
            <div className="flex flex-col items-center gap-2">
              <Loader2 className="h-8 w-8 text-[var(--accent-primary)] animate-spin" />
              <p className="text-[var(--text-secondary)]">上傳中...</p>
            </div>
          ) : preview ? (
            <div className="relative inline-block">
              <img src={preview} alt="Poster preview" className="max-h-48 mx-auto rounded" />
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  clearPreview();
                }}
                className="absolute -top-2 -right-2 p-1 bg-[var(--error)] text-white rounded-full hover:bg-[var(--error)] transition-colors"
                aria-label="清除預覽"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-2 text-[var(--text-secondary)]">
              <Image className="h-10 w-10" />
              <p>拖放圖片或點擊選擇檔案</p>
              <p className="text-xs text-[var(--text-muted)]">支援 JPG、PNG、WebP，最大 5MB</p>
            </div>
          )}
        </div>
      ) : (
        <div className="flex gap-2">
          <input
            type="url"
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
            placeholder="輸入海報圖片網址"
            className={cn(
              'flex-1 px-4 py-2',
              'bg-[var(--bg-secondary)] border border-[var(--border-subtle)] rounded-lg',
              'text-white placeholder-[var(--text-muted)]',
              'focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)] focus:border-transparent',
              'transition-colors'
            )}
            data-testid="poster-url-input"
          />
          <button
            type="button"
            onClick={handleUrlSubmit}
            disabled={!urlInput.trim()}
            className={cn(
              'px-4 py-2 rounded-lg bg-[var(--accent-primary)] text-white',
              'hover:bg-[var(--accent-pressed)] transition-colors',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            套用
          </button>
        </div>
      )}

      {/* Error Display */}
      {(validationError || error) && (
        <p className="text-sm text-[var(--error)]">{validationError || error}</p>
      )}
    </div>
  );
}

export default PosterUploader;
