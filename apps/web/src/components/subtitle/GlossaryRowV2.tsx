// Implements: Component/GlossaryRow-v2 (nDSEd)
// Source: ux-design.pen (Pencil app)
/**
 * One glossary pair row (ux3-subtitle-v2 AC 4, Component Library cell Fx24g):
 * `term_src ↔ term_zh`, source badge（字幕/中繼資料/手動）, unconfirmed visually
 * distinct（未確認 warning badge + confirm action）, row actions edit/confirm/
 * delete. The destructive delete is gated behind a Radix Dialog confirm (AC 7).
 * term_src is Latin-ish source text → Mono; term_zh is zh-TW → Noto (DL-v2 font
 * split). Only `term_zh`/`confirmed` are editable (PUT contract).
 */
import { useState } from 'react';
import { ArrowRight, Trash2 } from 'lucide-react';
import { Dialog, DialogContent, DialogTitle, DialogDescription, DialogFooter } from '../ui/Dialog';
import { cn } from '../../lib/utils';
import type { GlossaryTerm, GlossarySource } from '../../services/glossaryService';

const SOURCE_BADGE: Record<GlossarySource, { label: string; className: string }> = {
  subtitle: { label: '字幕', className: 'bg-[var(--info-tint)] text-[var(--info)]' },
  metadata: { label: '中繼資料', className: 'bg-[var(--accent-tint)] text-[var(--accent-text)]' },
  manual: { label: '手動', className: 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]' },
};

export interface GlossaryRowV2Props {
  term: GlossaryTerm;
  onConfirm: (termId: string) => void;
  /** PUT {term_zh, confirmed} — zh text is the only editable field. */
  onEdit: (termId: string, termZh: string) => void;
  onDelete: (termId: string) => void;
  /** Disables actions while a mutation is in flight. */
  busy?: boolean;
}

export function GlossaryRowV2({
  term,
  onConfirm,
  onEdit,
  onDelete,
  busy = false,
}: GlossaryRowV2Props) {
  const [editing, setEditing] = useState(false);
  const [draftZh, setDraftZh] = useState(term.termZh);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);

  const badge = SOURCE_BADGE[term.source] ?? SOURCE_BADGE.manual;

  const saveEdit = () => {
    const next = draftZh.trim();
    if (next && next !== term.termZh) onEdit(term.id, next);
    setEditing(false);
  };

  return (
    <div
      data-testid={`glossary-row-${term.id}`}
      className="flex min-h-[54px] items-center gap-3 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3.5 py-1.5"
    >
      <span className="shrink-0 font-mono text-[13px] text-[var(--text-primary)]">
        {term.termSrc}
      </span>
      <ArrowRight className="h-3.5 w-3.5 shrink-0 text-[var(--text-muted)]" aria-hidden="true" />
      {editing ? (
        <input
          type="text"
          value={draftZh}
          onChange={(e) => setDraftZh(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') saveEdit();
            if (e.key === 'Escape') {
              setDraftZh(term.termZh);
              setEditing(false);
            }
          }}
          aria-label={`編輯 ${term.termSrc} 的譯名`}
          data-testid={`glossary-edit-input-${term.id}`}
          /* eslint-disable-next-line jsx-a11y/no-autofocus -- edit mode is user-initiated; focus follows the action */
          autoFocus
          className="w-32 rounded-[var(--radius-sm)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] px-2 py-1 text-[13px] text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none"
        />
      ) : (
        <span className="truncate text-[13px] text-[var(--text-primary)]">{term.termZh}</span>
      )}

      <span className="min-w-0 flex-1" />

      <span
        className={cn(
          'shrink-0 rounded-[var(--radius-sm)] px-2 py-0.5 text-[11px]',
          badge.className
        )}
      >
        {badge.label}
      </span>

      {!term.confirmed && (
        <span
          data-testid={`glossary-unconfirmed-${term.id}`}
          className="shrink-0 rounded-[var(--radius-sm)] bg-[var(--warning-tint)] px-2 py-0.5 text-[11px] text-[var(--warning)]"
        >
          未確認
        </span>
      )}

      {editing ? (
        <>
          <button
            type="button"
            onClick={saveEdit}
            disabled={busy}
            data-testid={`glossary-save-${term.id}`}
            className="flex min-h-[44px] shrink-0 items-center px-2.5 text-[13px] font-semibold text-[var(--accent-text)] hover:text-[var(--accent-hover)] disabled:opacity-50"
          >
            儲存
          </button>
          <button
            type="button"
            onClick={() => {
              setDraftZh(term.termZh);
              setEditing(false);
            }}
            className="flex min-h-[44px] shrink-0 items-center px-2.5 text-[13px] text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          >
            取消
          </button>
        </>
      ) : (
        <>
          {!term.confirmed && (
            <button
              type="button"
              onClick={() => onConfirm(term.id)}
              disabled={busy}
              data-testid={`glossary-confirm-${term.id}`}
              className="flex min-h-[44px] shrink-0 items-center px-2.5 text-[13px] font-semibold text-[var(--accent-text)] hover:text-[var(--accent-hover)] disabled:opacity-50"
            >
              確認
            </button>
          )}
          <button
            type="button"
            onClick={() => {
              setDraftZh(term.termZh);
              setEditing(true);
            }}
            disabled={busy}
            data-testid={`glossary-edit-${term.id}`}
            className="flex min-h-[44px] shrink-0 items-center px-2.5 text-[13px] text-[var(--text-secondary)] hover:text-[var(--text-primary)] disabled:opacity-50"
          >
            編輯
          </button>
          <button
            type="button"
            onClick={() => setConfirmDeleteOpen(true)}
            disabled={busy}
            aria-label={`刪除 ${term.termSrc}`}
            data-testid={`glossary-delete-${term.id}`}
            className="flex min-h-[44px] w-9 shrink-0 items-center justify-center text-[var(--text-muted)] hover:text-[var(--error-text)] disabled:opacity-50"
          >
            <Trash2 className="h-4 w-4" aria-hidden="true" />
          </button>
        </>
      )}

      {/* Destructive confirm — Radix Dialog (AC 7). */}
      <Dialog open={confirmDeleteOpen} onOpenChange={setConfirmDeleteOpen}>
        <DialogContent data-testid={`glossary-delete-dialog-${term.id}`}>
          <DialogTitle>刪除詞彙</DialogTitle>
          <DialogDescription>
            確定要刪除「{term.termSrc} → {term.termZh}」嗎？此操作無法復原。
          </DialogDescription>
          <DialogFooter>
            <button
              type="button"
              onClick={() => setConfirmDeleteOpen(false)}
              className="flex min-h-[44px] items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm text-[var(--text-primary)]"
            >
              取消
            </button>
            <button
              type="button"
              onClick={() => {
                setConfirmDeleteOpen(false);
                onDelete(term.id);
              }}
              data-testid={`glossary-delete-confirm-${term.id}`}
              className="flex min-h-[44px] items-center justify-center rounded-[var(--radius-md)] bg-[var(--error)] px-4 text-sm font-medium text-[var(--text-on-accent)]"
            >
              刪除
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
