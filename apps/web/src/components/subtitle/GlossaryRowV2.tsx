// Implements: Component/GlossaryRow-v2 (nDSEd) + Component/GlossaryRow-v2/Mobile (t28C2s)
// Source: ux-design.pen (Pencil app)
/**
 * One glossary pair row (ux3-subtitle-v2 AC 4, Component Library cell Fx24g):
 * `term_src ↔ term_zh`, source badge（字幕/中繼資料/手動/官方字幕/社群 — one
 * neutral pill for all five, dsr-6c）, unconfirmed visually distinct（未確認
 * warning badge + confirm action）, row actions edit/confirm/delete — delete is
 * a text button (nDSEd Q11NpX), gated behind a Radix Dialog confirm (AC 7).
 * term_src is Latin-ish source text → Mono; term_zh is zh-TW → Noto (DL-v2 font
 * split). Only `term_zh`/`confirmed` are editable (PUT contract).
 */
import { useState, type KeyboardEvent } from 'react';
import { ArrowRight } from 'lucide-react';
import { Dialog, DialogContent, DialogTitle, DialogDescription, DialogFooter } from '../ui/Dialog';
import { MOBILE_SHEET_CLOSE, MOBILE_SHEET_CONTENT, SheetGrabber } from '../ui/mobileSheet';
import { cn } from '../../lib/utils';
import type { GlossaryTerm, GlossarySource } from '../../services/glossaryService';
import { isImeComposing } from '../../utils/keyboard';

// A source is a property of the term, not something that happened, so it never
// wears a state colour (DESIGN.md:300; TechBadge ruling 2026-09-10; dsr-6c).
// The label alone tells the five apart.
const SOURCE_LABEL: Record<GlossarySource, string> = {
  subtitle: '字幕',
  metadata: '中繼資料',
  manual: '手動',
  // sub-7-1 AC #4 (P1-10 殘餘): the two provenances the shared drawer brings.
  official_subtitle: '官方字幕',
  community: '社群',
};

const BADGE_SHAPE = 'shrink-0 rounded-full px-2.5 py-1 text-[11px]';

export interface GlossaryRowV2Props {
  term: GlossaryTerm;
  onConfirm: (termId: string) => void;
  /**
   * PUT {term_zh, confirmed} — zh text is the only editable field. Resolves when
   * the save landed; rejects when it did not, and the row then stays in edit
   * mode with the typed text (dsr-6c AC #5).
   */
  onEdit: (termId: string, termZh: string) => Promise<void>;
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
  // True while this row's own save is on its way. 取消 and Esc wait for it:
  // cancelling would throw away the text that a failed save must hand back.
  const [saving, setSaving] = useState(false);

  const sourceLabel = SOURCE_LABEL[term.source] ?? SOURCE_LABEL.manual;

  const cancelEdit = () => {
    if (saving) return;
    setDraftZh(term.termZh);
    setEditing(false);
  };

  const saveEdit = async () => {
    if (busy || saving) return;
    const next = draftZh.trim();
    if (!next || next === term.termZh) {
      setEditing(false);
      return;
    }
    setSaving(true);
    try {
      await onEdit(term.id, next);
      setEditing(false);
    } catch {
      // The save did not land: stay in edit mode so the typed text survives.
      // The panel shows why (glossary-write-error).
    } finally {
      setSaving(false);
    }
  };

  // Esc from anywhere in the editor (input, 儲存, 取消) cancels the edit. The
  // panel's onEscapeKeyDown keeps that same Esc from closing the whole panel
  // (data-glossary-inline-editor below). An Enter or Esc that only picks or
  // drops an input-method candidate is ignored.
  const onEditorKeyDown = (e: KeyboardEvent<HTMLElement>) => {
    if (isImeComposing(e)) return;
    if (e.key === 'Escape') cancelEdit();
  };

  return (
    <div
      data-testid={`glossary-row-${term.id}`}
      data-glossary-inline-editor={editing ? '' : undefined}
      className="flex min-h-[54px] items-center gap-3 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3.5 py-1.5 max-sm:flex-col max-sm:items-stretch max-sm:gap-2"
    >
      {/* Two wrappers, both sm:contents — on a desktop they dissolve and the row
          is the same single-line flex it has always been; on a phone they are
          the two lines the design asks for (F6-M-v2): the pair on top, badges
          and actions underneath. A phone cannot fit all eight children on one
          line at 390 once 編輯 is back on every row (⚖️ 2026-09-17). */}
      <div
        data-testid={`glossary-row-line1-${term.id}`}
        // max-sm:flex-wrap is line 1's safety net, the same one line 2 has: a
        // long term_src (subtitle/metadata extraction produces multi-word proper
        // nouns) drops 譯名 to its own line rather than running off a 328px
        // phone. shrink + truncate is the last resort under that, for a single
        // unbreakable token wider than the whole line.
        className="flex min-w-0 flex-wrap items-center gap-3 sm:contents"
      >
        <span className="shrink-0 truncate font-mono text-sm text-[var(--text-primary)] max-sm:min-w-0 max-sm:shrink">
          {term.termSrc}
        </span>
        <ArrowRight className="h-3.5 w-3.5 shrink-0 text-[var(--text-muted)]" aria-hidden="true" />
        {editing ? (
          <input
            type="text"
            value={draftZh}
            onChange={(e) => setDraftZh(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !isImeComposing(e)) {
                void saveEdit();
                return;
              }
              onEditorKeyDown(e);
            }}
            aria-label={`編輯 ${term.termSrc} 的譯名`}
            data-testid={`glossary-edit-input-${term.id}`}
            /* eslint-disable-next-line jsx-a11y/no-autofocus -- edit mode is user-initiated; focus follows the action */
            autoFocus
            className="w-32 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none"
          />
        ) : (
          <span className="truncate text-sm text-[var(--text-primary)]">{term.termZh}</span>
        )}
      </div>

      <div
        data-testid={`glossary-row-line2-${term.id}`}
        className="flex flex-wrap items-center gap-3 sm:contents"
      >
        <span aria-hidden="true" className="min-w-0 flex-1 max-sm:hidden" />

        <span
          data-testid={`glossary-source-${term.id}`}
          className={`${BADGE_SHAPE} bg-[var(--bg-tertiary)] text-[var(--text-secondary)]`}
        >
          {sourceLabel}
        </span>

        {!term.confirmed && (
          <span
            data-testid={`glossary-unconfirmed-${term.id}`}
            className={`${BADGE_SHAPE} bg-[var(--warning-tint)] text-[var(--warning-text)]`}
          >
            未確認
          </span>
        )}

        {/* The desktop spacer above sits BEFORE the badges; on a phone the badges
          belong on the left and the actions on the right, so line 2 needs its
          own spacer here. One per breakpoint, never both visible. */}
        <span aria-hidden="true" className="min-w-0 flex-1 sm:hidden" />

        {editing ? (
          <>
            <button
              type="button"
              onClick={() => void saveEdit()}
              onKeyDown={onEditorKeyDown}
              // aria-disabled, not disabled: a button that turns disabled under
              // focus drops focus to <body>, and an Esc from there would close
              // the whole panel mid-save. saveEdit ignores clicks while busy.
              aria-disabled={busy || saving}
              data-testid={`glossary-save-${term.id}`}
              className="flex min-h-[44px] shrink-0 items-center px-2.5 text-sm font-semibold text-[var(--accent-text)] hover:underline aria-disabled:opacity-50"
            >
              儲存
            </button>
            <button
              type="button"
              onClick={cancelEdit}
              onKeyDown={onEditorKeyDown}
              aria-disabled={saving}
              data-testid={`glossary-cancel-${term.id}`}
              className="flex min-h-[44px] shrink-0 items-center px-2.5 text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] aria-disabled:opacity-50"
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
                aria-label={`確認 ${term.termSrc}`}
                data-testid={`glossary-confirm-${term.id}`}
                className="flex min-h-[44px] shrink-0 items-center px-2.5 text-sm font-semibold text-[var(--accent-text)] hover:underline disabled:opacity-50"
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
              aria-label={`編輯 ${term.termSrc}`}
              data-testid={`glossary-edit-${term.id}`}
              className="flex min-h-[44px] shrink-0 items-center px-2.5 text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] disabled:opacity-50"
            >
              編輯
            </button>
            <button
              type="button"
              onClick={() => setConfirmDeleteOpen(true)}
              disabled={busy}
              aria-label={`刪除 ${term.termSrc}`}
              data-testid={`glossary-delete-${term.id}`}
              className="flex min-h-[44px] shrink-0 items-center px-2.5 text-sm text-[var(--error-text)] hover:underline disabled:opacity-50"
            >
              刪除
            </button>
          </>
        )}
      </div>

      {/* Destructive confirm — Radix Dialog (AC 7), shaped like Component/DialogFrame (m6KMPr).
          On a phone it is the shared bottom sheet (dsr-6f-1); the desktop width
          AND radius are sm:-prefixed or twMerge drops the sheet's own corners. */}
      <Dialog open={confirmDeleteOpen} onOpenChange={setConfirmDeleteOpen}>
        <DialogContent
          data-testid={`glossary-delete-dialog-${term.id}`}
          closeClassName={cn(MOBILE_SHEET_CLOSE, 'max-sm:top-4')}
          className={cn(
            'flex flex-col gap-4',
            MOBILE_SHEET_CONTENT,
            'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:max-w-[480px] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)]'
          )}
        >
          <SheetGrabber data-testid="glossary-delete-sheet-grabber" />
          <div className="flex flex-col gap-1">
            {/* This sheet has no title ROW to centre the ✕ against — it sits
                straight over the content — so the title keeps the same 48px
                clearance the other sheets get from their header's pr-12. */}
            <DialogTitle className="text-xl max-sm:pr-12">刪除詞彙</DialogTitle>
            <DialogDescription>
              確定要刪除「{term.termSrc} → {term.termZh}」嗎？此操作無法復原。
            </DialogDescription>
          </div>
          <DialogFooter>
            <button
              type="button"
              onClick={() => setConfirmDeleteOpen(false)}
              className="flex min-h-[44px] items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)]"
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
              className="flex min-h-[44px] items-center justify-center rounded-[var(--radius-md)] bg-[var(--error)] px-5 text-sm font-semibold text-[var(--text-on-scrim)]"
            >
              刪除
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
