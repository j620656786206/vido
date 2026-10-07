// Design ref: ux-design.pen Screen C9-D (NR3zK) · C9-M (AWYm0) · C9-D Note (JTw7U)
/**
 * 從官方字幕學譯名 — sub-7-5b AC #3.
 *
 * When some episodes of a show already carry an official zh-Hant subtitle and
 * others do not, Vido reads the ones that do and learns how the show's names
 * are rendered, then the paid translations of the rest use the same names.
 * It runs by itself after a scan; this section is the manual re-run and the
 * place to see what the last run learned.
 *
 * $0 local work, so a plain Secondary button — never ButtonCost (J9 keeps the
 * money mark for buttons that spend). The result is one line left in place,
 * not a toast: "what did it learn last time" is something you come back to
 * read (C9-D note ①–④).
 */
import { AlertTriangle } from 'lucide-react';
import { Button } from '../ui/Button';
import { cn } from '../../lib/utils';
import { useGlossaryMineStatus, useStartGlossaryMine } from '../../hooks/useGlossaryMine';
import type { MineResult, MineStatus } from '../../services/glossaryMineService';

/** 「2026-10-01 14:30」 in the viewer's local time. */
export function formatRunTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** One show's sentence (note ③): what it learned, and any fan-group files skipped. */
export function describeResult(r: MineResult): string {
  if (r.error) return `${r.title || r.seriesId} 沒學成`;
  const skipped = r.fansubSkipped > 0 ? `跳過 ${r.fansubSkipped} 個字幕組檔` : '';
  if (r.episodesUsed === 0) {
    return skipped ? `${r.title} ${skipped}` : `${r.title} 沒有可用的集`;
  }
  const replaced =
    r.termsReplaced && r.termsReplaced > 0 ? `，其中 ${r.termsReplaced} 個修正了聽寫猜的寫法` : '';
  const learned = `${r.title} 學到 ${r.termsFound} 個詞（${r.episodesUsed} 集）${replaced}`;
  return skipped ? `${learned}，${skipped}` : learned;
}

/** The status line under/next to the button: notes ①, ③, ④. */
export function statusLine(status: MineStatus | undefined): string {
  if (!status?.lastRunAt) return '還沒跑過';
  const when = `上次 ${formatRunTime(status.lastRunAt)}`;
  if (status.results.length === 0) return `${when} · 片庫裡沒有「部分集有官方字幕」的影集`;
  return `${when} · ${status.results.map(describeResult).join('、')}`;
}

export function OfficialSubtitleMiningCard() {
  const { data: status, isError } = useGlossaryMineStatus();
  const start = useStartGlossaryMine();
  const running = !!status?.running || start.isPending;

  return (
    <section
      data-testid="official-subtitle-mining"
      aria-labelledby="official-subtitle-mining-title"
      className="mt-10 max-w-3xl"
    >
      <h2
        id="official-subtitle-mining-title"
        className="mb-1 text-base font-semibold text-[var(--text-primary)]"
      >
        從官方字幕學譯名
      </h2>
      <p className="mb-3 text-sm text-[var(--text-secondary)]">
        同一部劇有些集已經有官方繁中字幕、有些集沒有時，Vido
        會讀那幾集學會這部劇的人名怎麼翻，再用同樣的譯名翻其他集。掃描完會自動做；這裡可以手動再跑一次。
      </p>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <Button
          type="button"
          variant="secondary"
          data-testid="official-subtitle-mining-start"
          disabled={running}
          aria-busy={running || undefined}
          onClick={() => start.mutate()}
          className="min-h-[44px] w-full shrink-0 max-sm:bg-[var(--accent-primary)] max-sm:text-[var(--text-on-accent)] sm:min-h-0 sm:w-auto"
        >
          {running ? '學習中…' : '重新從官方字幕學習'}
        </Button>
        <p
          data-testid="official-subtitle-mining-status"
          aria-live="polite"
          className={cn('text-sm text-[var(--text-secondary)]')}
        >
          {running ? '正在讀各集字幕與抽軌，不花錢；可以離開這頁。' : statusLine(status)}
        </p>
      </div>

      {(isError || start.isError) && (
        <p
          role="alert"
          data-testid="official-subtitle-mining-error"
          className="mt-3 flex items-center gap-2 text-sm text-[var(--error-text)]"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
          {start.error?.message ?? '無法讀取上次的結果。'}
        </p>
      )}
    </section>
  );
}
