import { createFileRoute } from '@tanstack/react-router';
import { Upload } from 'lucide-react';
import { MetadataExport } from '../../components/settings/MetadataExport';

export const Route = createFileRoute('/settings/export')({
  component: ExportSettingsPage,
});

// 匯出 shipped long ago inside 備份與還原 — a working exporter hidden behind a
// tab that claimed「尚未實作」was the strip's one outright lie (critique R1,
// graduated R4). It now owns this page; 匯入 states its pending half honestly
// below instead of the whole tab pretending neither exists.
function ExportSettingsPage() {
  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">匯出/匯入</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        將媒體庫元資料匯出為 JSON、YAML 或 NFO 檔案。
      </p>
      <div className="space-y-6">
        <MetadataExport />
        <div
          className="flex items-center gap-2 rounded-lg border border-dashed border-[var(--border-subtle)] p-4 text-sm text-[var(--text-muted)]"
          data-testid="import-pending"
        >
          <Upload className="h-4 w-4 shrink-0" aria-hidden="true" />
          匯入功能尚未實作 — 目前請以「備份與還原」還原完整資料。
        </div>
      </div>
    </div>
  );
}
