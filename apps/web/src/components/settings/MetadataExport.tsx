import { useState } from 'react';
import { FileDown, Loader2, Download } from 'lucide-react';
import { useExport } from '../../hooks/useBackups';
import { backupService } from '../../services/backupService';

type ExportFormat = 'json' | 'yaml' | 'nfo';

const FORMAT_OPTIONS: { value: ExportFormat; label: string; description: string }[] = [
  { value: 'json', label: 'JSON', description: '人類可讀的 JSON 格式，適合程式處理' },
  { value: 'yaml', label: 'YAML', description: '人類可讀的 YAML 格式，適合設定備份' },
  { value: 'nfo', label: 'NFO', description: 'Kodi/Plex/Jellyfin 相容的 .nfo 檔案' },
];

export function MetadataExport() {
  const exportMutation = useExport();
  const [format, setFormat] = useState<ExportFormat>('json');
  const [message, setMessage] = useState<string | null>(null);
  const [downloadId, setDownloadId] = useState<string | null>(null);

  const handleExport = async () => {
    if (exportMutation.isPending) return;
    setMessage(null);
    setDownloadId(null);
    try {
      const result = await exportMutation.mutateAsync(format);
      if (result.status === 'completed') {
        setMessage(`✅ ${result.message}`);
        if (result.exportId && result.format !== 'nfo') {
          setDownloadId(result.exportId);
        }
      } else {
        setMessage(`⚠️ 匯出失敗：${result.error || '未知錯誤'}`);
      }
    } catch (err) {
      setMessage(err instanceof Error ? err.message : '匯出失敗');
    }
  };

  return (
    <div
      className="rounded-lg border border-slate-700 bg-slate-800 p-4"
      data-testid="metadata-export"
    >
      <div className="flex items-center gap-2 mb-4">
        <FileDown className="h-4 w-4 text-slate-400" />
        <span className="text-sm font-medium text-slate-200">匯出媒體資料</span>
      </div>

      <div className="space-y-3">
        {/* Format selector */}
        <div className="space-y-2">
          {FORMAT_OPTIONS.map((opt) => (
            <label
              key={opt.value}
              className={`flex cursor-pointer items-center gap-3 rounded-lg border p-3 transition-colors ${
                format === opt.value
                  ? 'border-blue-500 bg-blue-500/10'
                  : 'border-slate-700 hover:border-slate-600'
              }`}
              data-testid={`export-format-${opt.value}`}
            >
              <input
                type="radio"
                name="exportFormat"
                value={opt.value}
                checked={format === opt.value}
                onChange={() => setFormat(opt.value)}
                className="accent-blue-500"
              />
              <div>
                <span className="text-sm font-medium text-slate-200">{opt.label}</span>
                <p className="text-xs text-slate-500">{opt.description}</p>
              </div>
            </label>
          ))}
        </div>

        {/* Export button */}
        <button
          onClick={handleExport}
          disabled={exportMutation.isPending}
          className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:opacity-50"
          data-testid="export-btn"
        >
          {exportMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <FileDown className="h-4 w-4" />
          )}
          匯出
        </button>

        {/* Message */}
        {message && (
          <p className="text-xs text-slate-400" data-testid="export-message">
            {message}
          </p>
        )}

        {/* Download link */}
        {downloadId && (
          <a
            href={backupService.getExportDownloadUrl(downloadId)}
            className="inline-flex items-center gap-1 text-xs text-blue-400 hover:text-blue-300"
            data-testid="export-download-link"
          >
            <Download className="h-3 w-3" />
            下載匯出檔案
          </a>
        )}
      </div>
    </div>
  );
}
