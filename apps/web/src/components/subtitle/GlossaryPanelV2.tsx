// Design ref: ux-design.pen Screen F6-D-v2 (dlfMR)
/**
 * Glossary management + review panel (ux3-subtitle-v2 AC 4, screens F6-D-v2
 * dlfMR / F6-M-v2 buepS / F7-D-v2 A85GFD 空狀態). List / add / edit / per-row
 * confirm / delete / 全部確認 batch-confirm (party-mode P1) over the six 9R-15
 * routes. Four-state coverage (AC 5): loading skeleton, empty (尚無詞彙 —
 * 生成字幕時自動累積, distinct from failure), fail-soft error + 重試, default.
 * Rule 5: list = query, writes = mutations (useGlossary.ts).
 */
import { useState } from 'react';
import { BookOpen, CircleAlert, Plus, CheckCheck } from 'lucide-react';
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '../ui/Dialog';
import { useGlossaryTerms, useGlossaryMutations } from '../../hooks/useGlossary';
import { GlossaryRowV2 } from './GlossaryRowV2';

export interface GlossaryPanelV2Props {
  /** STRING local media id (9R-15 route contract). */
  mediaId: string;
  mediaTitle: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function SkeletonRows() {
  return (
    <div data-testid="glossary-loading" className="flex flex-col gap-2" aria-hidden="true">
      {[0, 1, 2].map((i) => (
        <div
          key={i}
          className="h-[54px] animate-pulse rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] motion-reduce:animate-none"
        />
      ))}
    </div>
  );
}

export function GlossaryPanelV2({ mediaId, mediaTitle, open, onOpenChange }: GlossaryPanelV2Props) {
  const terms = useGlossaryTerms(mediaId, open);
  const { add, edit, confirm, confirmAll, remove } = useGlossaryMutations(mediaId);
  const [adding, setAdding] = useState(false);
  const [draftSrc, setDraftSrc] = useState('');
  const [draftZh, setDraftZh] = useState('');

  const list = terms.data ?? [];
  const unconfirmedCount = list.filter((t) => !t.confirmed).length;
  const busy =
    add.isPending ||
    edit.isPending ||
    confirm.isPending ||
    confirmAll.isPending ||
    remove.isPending;

  const submitAdd = () => {
    const termSrc = draftSrc.trim();
    const termZh = draftZh.trim();
    if (!termSrc || !termZh) return;
    // UI-added terms are manual + confirmed (the user just typed the mapping).
    add.mutate(
      { termSrc, termZh, source: 'manual', confirmed: true },
      {
        onSuccess: () => {
          setDraftSrc('');
          setDraftZh('');
          setAdding(false);
        },
      }
    );
  };

  const addForm = adding && (
    <div
      data-testid="glossary-add-form"
      className="flex items-center gap-2 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3.5 py-2"
    >
      <input
        type="text"
        value={draftSrc}
        onChange={(e) => setDraftSrc(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && submitAdd()}
        placeholder="原文（例：Demogorgon）"
        aria-label="原文詞彙"
        data-testid="glossary-add-src"
        className="w-40 rounded-[var(--radius-sm)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] px-2 py-1.5 font-mono text-[13px] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none"
      />
      <input
        type="text"
        value={draftZh}
        onChange={(e) => setDraftZh(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && submitAdd()}
        placeholder="譯名（例：魔王獸）"
        aria-label="中文譯名"
        data-testid="glossary-add-zh"
        className="w-40 rounded-[var(--radius-sm)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] px-2 py-1.5 text-[13px] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none"
      />
      <span className="flex-1" />
      <button
        type="button"
        onClick={submitAdd}
        disabled={busy || !draftSrc.trim() || !draftZh.trim()}
        data-testid="glossary-add-submit"
        className="flex min-h-[44px] items-center px-2.5 text-[13px] font-semibold text-[var(--accent-text)] disabled:opacity-50"
      >
        新增
      </button>
      <button
        type="button"
        onClick={() => setAdding(false)}
        className="flex min-h-[44px] items-center px-2.5 text-[13px] text-[var(--text-secondary)]"
      >
        取消
      </button>
    </div>
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        data-testid="glossary-panel-v2"
        className="flex max-h-[85vh] w-[calc(100vw-2rem)] max-w-3xl flex-col gap-0 p-0"
      >
        {/* Title bar */}
        <div className="flex h-14 items-center justify-between border-b border-[var(--border-subtle)] pl-6 pr-12">
          <DialogTitle className="text-base font-semibold">名詞對照表 — {mediaTitle}</DialogTitle>
        </div>

        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <DialogDescription className="text-[13px]">
              生成字幕時會依此表固定譯名
            </DialogDescription>
            <span className="hidden flex-1 sm:block" />
            {list.length > 0 && (
              <button
                type="button"
                onClick={() => confirmAll.mutate()}
                disabled={busy || unconfirmedCount === 0}
                data-testid="glossary-confirm-all"
                className="flex min-h-[44px] items-center justify-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] disabled:opacity-50"
              >
                <CheckCheck className="h-4 w-4" aria-hidden="true" />
                全部確認
              </button>
            )}
            {(list.length > 0 || terms.isError) && (
              <button
                type="button"
                onClick={() => setAdding(true)}
                disabled={busy}
                data-testid="glossary-add-term"
                className="flex min-h-[44px] items-center justify-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] disabled:opacity-50"
              >
                <Plus className="h-4 w-4" aria-hidden="true" />
                新增詞彙
              </button>
            )}
          </div>

          {addForm}

          {/* Four states: loading / error / empty / list */}
          {terms.isLoading ? (
            <SkeletonRows />
          ) : terms.isError ? (
            <div
              data-testid="glossary-error"
              className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3"
            >
              <CircleAlert className="h-4 w-4 shrink-0 text-[var(--error)]" aria-hidden="true" />
              <p className="flex-1 text-[13px] text-[var(--error-text)]">名詞對照表載入失敗</p>
              <button
                type="button"
                onClick={() => terms.refetch()}
                data-testid="glossary-retry"
                className="flex min-h-[44px] shrink-0 items-center px-3 text-[13px] font-semibold text-[var(--accent-text)]"
              >
                重試
              </button>
            </div>
          ) : list.length === 0 ? (
            <div data-testid="glossary-empty" className="flex flex-col items-center gap-3 py-12">
              <BookOpen className="h-10 w-10 text-[var(--text-muted)]" aria-hidden="true" />
              <p className="text-base font-semibold text-[var(--text-primary)]">尚無詞彙</p>
              <p className="text-[13px] text-[var(--text-secondary)]">生成字幕時自動累積</p>
              {!adding && (
                <button
                  type="button"
                  onClick={() => setAdding(true)}
                  data-testid="glossary-empty-add"
                  className="mt-2 flex min-h-[44px] items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
                >
                  <Plus className="h-4 w-4" aria-hidden="true" />
                  新增詞彙
                </button>
              )}
            </div>
          ) : (
            <div className="flex flex-col gap-2" data-testid="glossary-list">
              {list.map((term) => (
                <GlossaryRowV2
                  key={term.id}
                  term={term}
                  busy={busy}
                  onConfirm={(termId) => confirm.mutate(termId)}
                  onEdit={(termId, termZh) =>
                    edit.mutate({
                      termId,
                      termZh,
                      confirmed: list.find((t) => t.id === termId)?.confirmed ?? false,
                    })
                  }
                  onDelete={(termId) => remove.mutate(termId)}
                />
              ))}
            </div>
          )}
        </div>

        {/* Footer count — numbers Mono, zh Noto (DL-v2 font split). */}
        {list.length > 0 && (
          <div
            data-testid="glossary-footer-count"
            className="border-t border-[var(--border-subtle)] px-6 py-3.5 text-[13px] text-[var(--text-secondary)]"
          >
            共 <span className="font-mono tabular-nums">{list.length}</span> 條 ·{' '}
            <span className="font-mono tabular-nums">{unconfirmedCount}</span> 條未確認
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
