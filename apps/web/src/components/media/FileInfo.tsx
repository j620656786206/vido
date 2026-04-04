export interface FileInfoProps {
  filePath?: string;
  fileSize?: number;
}

export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export function parseQuality(filePath: string): string | null {
  const match = filePath.match(/\b(2160p|4K|1080p|720p|480p|360p)\b/i);
  return match ? match[1].toUpperCase().replace('2160P', '4K') : null;
}

function truncateFilename(filePath: string, maxLen = 40): string {
  const filename = filePath.split('/').pop() || filePath;
  if (filename.length <= maxLen) return filename;
  const ext = filename.lastIndexOf('.');
  if (ext > 0 && filename.length - ext < 8) {
    const name = filename.slice(0, ext);
    const extension = filename.slice(ext);
    return name.slice(0, maxLen - extension.length - 3) + '...' + extension;
  }
  return filename.slice(0, maxLen - 3) + '...';
}

export function FileInfo({ filePath, fileSize }: FileInfoProps) {
  if (!filePath && !fileSize) return null;

  const quality = filePath ? parseQuality(filePath) : null;
  const filename = filePath ? truncateFilename(filePath) : null;

  return (
    <div className="space-y-1 text-sm" data-testid="file-info">
      {filename && (
        <div className="flex items-center gap-2 text-[var(--text-secondary)]" title={filePath}>
          <span className="flex-shrink-0">📁</span>
          <span className="truncate" data-testid="file-name">
            {filename}
          </span>
        </div>
      )}
      <div className="flex items-center gap-3 text-[var(--text-secondary)]">
        {fileSize != null && fileSize > 0 && (
          <span data-testid="file-size">{formatFileSize(fileSize)}</span>
        )}
        {quality && (
          <span
            className="rounded bg-[var(--bg-tertiary)] px-1.5 py-0.5 text-xs font-medium text-[var(--text-secondary)]"
            data-testid="file-quality"
          >
            {quality}
          </span>
        )}
      </div>
    </div>
  );
}
