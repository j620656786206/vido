import { createFileRoute } from '@tanstack/react-router';
import { Clock, FileText } from 'lucide-react';

export const Route = createFileRoute('/pending')({
  component: PendingPage,
});

function PendingPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Clock className="h-6 w-6 text-[var(--warning)]" />
        <h1 className="text-2xl font-bold text-white">待解析</h1>
      </div>

      <div className="rounded-xl bg-[var(--bg-secondary)]/50 p-8 text-center">
        <FileText className="mx-auto h-12 w-12 text-[var(--text-muted)]" />
        <p className="mt-4 text-lg text-[var(--text-secondary)]">尚未有待解析的媒體檔案</p>
        <p className="mt-2 text-sm text-[var(--text-muted)]">
          當有新的媒體檔案需要解析時，它們會顯示在這裡
        </p>
      </div>
    </div>
  );
}
