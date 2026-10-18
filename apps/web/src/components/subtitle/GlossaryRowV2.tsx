// Implements: Component/GlossaryRow-v2 (nDSEd) + Component/GlossaryRow-v2/Mobile (t28C2s)
// Implements: Component/GlossaryRow-v2/Seasons (QkRiY) + Component/GlossaryRow-v2/Seasons/Mobile (lcSkQ)
// Source: ux-design.pen (Pencil app)
/**
 * One glossary card (ux3-subtitle-v2 AC 4, Component Library cell Fx24g):
 * `term_src ↔ term_zh`, source badge（字幕/中繼資料/手動/官方字幕/社群 — one
 * neutral pill for all five, dsr-6c）, unconfirmed visually distinct（未確認
 * warning badge + confirm action）, row actions edit/confirm/delete — delete is
 * a text button (nDSEd Q11NpX), gated behind a Radix Dialog confirm (AC 7).
 * term_src is Latin-ish source text → Mono; term_zh is zh-TW → Noto (DL-v2 font
 * split). Only `term_zh`/`confirmed` are editable (PUT contract).
 *
 * disc-2026-10-glossary-season-variants-card-b (F6s-D-v2 mHJar): a term whose
 * official subtitles spell it differently in some season carries those seasons
 * as indented lines under it —「↳ 第 N 季官方寫法：○○」with their own 編輯／刪除,
 * no badge and no 確認: the card's 確認 confirms the seasons with it.
 */
import { useState, type KeyboardEvent, type ReactNode } from 'react';
import { ArrowRight, CornerDownRight } from 'lucide-react';
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

const BADGE_SHAPE = 'shrink-0 rounded-full px-2.5 py-1 text-xs';
const CARD_SHAPE =
  'rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3.5 py-1.5';
const ACTION = 'flex min-h-[44px] shrink-0 items-center px-2.5 text-sm';
const EDIT_ACTION = `${ACTION} text-[var(--text-secondary)] hover:text-[var(--text-primary)] disabled:opacity-50`;
const DELETE_ACTION = `${ACTION} text-[var(--error-text)] hover:underline disabled:opacity-50`;

/** Inline edit state shared by the card's main line and its season lines. */
function useTermEditor(
  term: GlossaryTerm,
  onEdit: (termId: string, termZh: string) => Promise<void>,
  busy: boolean
) {
  const [editing, setEditing] = useState(false);
  const [draftZh, setDraftZh] = useState(term.termZh);
  // True while this row's own save is on its way. 取消 and Esc wait for it:
  // cancelling would throw away the text that a failed save must hand back.
  const [saving, setSaving] = useState(false);

  const start = () => {
    setDraftZh(term.termZh);
    setEditing(true);
  };

  const cancel = () => {
    if (saving) return;
    setDraftZh(term.termZh);
    setEditing(false);
  };

  const save = async () => {
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
    if (e.key === 'Escape') cancel();
  };

  return { editing, draftZh, setDraftZh, saving, start, cancel, save, onEditorKeyDown };
}

type TermEditor = ReturnType<typeof useTermEditor>;

function EditInput({ term, editor }: { term: GlossaryTerm; editor: TermEditor }) {
  return (
    <input
      type="text"
      value={editor.draftZh}
      onChange={(e) => editor.setDraftZh(e.target.value)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' && !isImeComposing(e)) {
          void editor.save();
          return;
        }
        editor.onEditorKeyDown(e);
      }}
      aria-label={
        term.season !== undefined
          ? `編輯 ${term.termSrc} 第 ${term.season} 季的譯名`
          : `編輯 ${term.termSrc} 的譯名`
      }
      data-testid={`glossary-edit-input-${term.id}`}
      /* eslint-disable-next-line jsx-a11y/no-autofocus -- edit mode is user-initiated; focus follows the action */
      autoFocus
      className="w-32 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none"
    />
  );
}

function SaveCancel({
  term,
  editor,
  busy,
}: {
  term: GlossaryTerm;
  editor: TermEditor;
  busy: boolean;
}) {
  return (
    <>
      <button
        type="button"
        onClick={() => void editor.save()}
        onKeyDown={editor.onEditorKeyDown}
        // aria-disabled, not disabled: a button that turns disabled under
        // focus drops focus to <body>, and an Esc from there would close
        // the whole panel mid-save. save() ignores clicks while busy.
        aria-disabled={busy || editor.saving}
        data-testid={`glossary-save-${term.id}`}
        className={`${ACTION} font-semibold text-[var(--accent-text)] hover:underline aria-disabled:opacity-50`}
      >
        儲存
      </button>
      <button
        type="button"
        onClick={editor.cancel}
        onKeyDown={editor.onEditorKeyDown}
        aria-disabled={editor.saving}
        data-testid={`glossary-cancel-${term.id}`}
        className={`${ACTION} text-[var(--text-secondary)] hover:text-[var(--text-primary)] aria-disabled:opacity-50`}
      >
        取消
      </button>
    </>
  );
}

/**
 * Destructive confirm — Radix Dialog (AC 7), shaped like Component/DialogFrame (m6KMPr).
 * On a phone it is the shared bottom sheet (dsr-6f-1); the desktop width AND
 * radius are sm:-prefixed or twMerge drops the sheet's own corners.
 */
function DeleteTermDialog({
  term,
  open,
  onOpenChange,
  description,
  onDelete,
}: {
  term: GlossaryTerm;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  description: ReactNode;
  onDelete: (termId: string) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
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
          <DialogTitle className="text-lg sm:text-xl max-sm:pr-12">刪除詞彙</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </div>
        <DialogFooter>
          <button
            type="button"
            onClick={() => onOpenChange(false)}
            className="flex min-h-[44px] items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)]"
          >
            取消
          </button>
          <button
            type="button"
            onClick={() => {
              onOpenChange(false);
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
  );
}

interface SeasonLineProps {
  term: GlossaryTerm;
  /** The show-wide spelling this season falls back to once its line is deleted. */
  fallbackZh?: string;
  onEdit: (termId: string, termZh: string) => Promise<void>;
  onDelete: (termId: string) => void;
  busy: boolean;
}

/**
 * 「↳ 第 N 季官方寫法：○○」(gs-season / gsm-season). basis-full makes it its own
 * line inside the card's flex-wrap. Its 編輯／刪除 sit in a gap-3 pair with the
 * same right edge as the main line's, so the two action columns line up — the
 * 4px drift on the design is a Pencil gap side effect, not intent (Sally CR).
 */
function SeasonLine({ term, fallbackZh, onEdit, onDelete, busy }: SeasonLineProps) {
  const editor = useTermEditor(term, onEdit, busy);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const season = term.season ?? 0;

  return (
    <div
      data-testid={`glossary-season-row-${term.id}`}
      data-glossary-inline-editor={editor.editing ? '' : undefined}
      className="flex min-h-[44px] basis-full items-center gap-3 pl-6 max-sm:basis-auto max-sm:pl-2"
    >
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <CornerDownRight
          className="h-3.5 w-3.5 shrink-0 text-[var(--text-muted)]"
          aria-hidden="true"
        />
        <span className="shrink-0 text-sm text-[var(--text-secondary)]">
          第 {season} 季官方寫法：
        </span>
        {editor.editing ? (
          <EditInput term={term} editor={editor} />
        ) : (
          <span className="min-w-0 truncate text-sm text-[var(--text-primary)]">{term.termZh}</span>
        )}
      </div>
      <div className="flex shrink-0 items-center gap-3">
        {editor.editing ? (
          <SaveCancel term={term} editor={editor} busy={busy} />
        ) : (
          <>
            <button
              type="button"
              onClick={editor.start}
              disabled={busy}
              aria-label={`編輯 ${term.termSrc} 第 ${season} 季的寫法`}
              data-testid={`glossary-edit-${term.id}`}
              className={EDIT_ACTION}
            >
              編輯
            </button>
            <button
              type="button"
              onClick={() => setConfirmDeleteOpen(true)}
              disabled={busy}
              aria-label={`刪除 ${term.termSrc} 第 ${season} 季的寫法`}
              data-testid={`glossary-delete-${term.id}`}
              className={DELETE_ACTION}
            >
              刪除
            </button>
          </>
        )}
      </div>
      <DeleteTermDialog
        term={term}
        open={confirmDeleteOpen}
        onOpenChange={setConfirmDeleteOpen}
        onDelete={onDelete}
        description={
          fallbackZh
            ? `這是第 ${season} 季官方字幕的寫法。刪掉後，補第 ${season} 季的集數會改用全劇的寫法「${fallbackZh}」。此操作無法復原。`
            : `這是第 ${season} 季官方字幕的寫法，全劇層已經沒有這個詞。刪掉後「${term.termSrc}」在第 ${season} 季就不在名詞表裡了。此操作無法復原。`
        }
      />
    </div>
  );
}

export interface GlossaryRowV2Props {
  term: GlossaryTerm;
  /**
   * The season spellings this card shows under the term — only those that
   * differ from it (glossaryCards.groupGlossaryCards). Empty for most terms.
   */
  seasons?: GlossaryTerm[];
  /**
   * 未確認 for the whole card. Defaults to the term's own state; a card whose
   * season line is unconfirmed is unconfirmed too, and its 確認 confirms all.
   */
  unconfirmed?: boolean;
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
  seasons = [],
  unconfirmed = !term.confirmed,
  onConfirm,
  onEdit,
  onDelete,
  busy = false,
}: GlossaryRowV2Props) {
  const editor = useTermEditor(term, onEdit, busy);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);

  const sourceLabel = SOURCE_LABEL[term.source] ?? SOURCE_LABEL.manual;
  const seasonNote = seasons.map((s) => `第 ${s.season} 季的官方寫法「${s.termZh}」`).join('、');

  return (
    <div
      data-testid={`glossary-row-${term.id}`}
      data-glossary-inline-editor={editor.editing ? '' : undefined}
      // flex-wrap + gap-y-0: the season lines are basis-full, so each wraps
      // onto its own line right under the main one; a card without seasons
      // is the same single line it always was.
      className={`flex min-h-[54px] flex-wrap items-center gap-x-3 gap-y-0 ${CARD_SHAPE} max-sm:flex-col max-sm:flex-nowrap max-sm:items-stretch max-sm:gap-2`}
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
        {editor.editing ? (
          <EditInput term={term} editor={editor} />
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

        {unconfirmed && (
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

        {editor.editing ? (
          <SaveCancel term={term} editor={editor} busy={busy} />
        ) : (
          <>
            {unconfirmed && (
              <button
                type="button"
                onClick={() => onConfirm(term.id)}
                disabled={busy}
                aria-label={`確認 ${term.termSrc}`}
                data-testid={`glossary-confirm-${term.id}`}
                className={`${ACTION} font-semibold text-[var(--accent-text)] hover:underline disabled:opacity-50`}
              >
                確認
              </button>
            )}
            <button
              type="button"
              onClick={editor.start}
              disabled={busy}
              aria-label={`編輯 ${term.termSrc}`}
              data-testid={`glossary-edit-${term.id}`}
              className={EDIT_ACTION}
            >
              編輯
            </button>
            <button
              type="button"
              onClick={() => setConfirmDeleteOpen(true)}
              disabled={busy}
              aria-label={`刪除 ${term.termSrc}`}
              data-testid={`glossary-delete-${term.id}`}
              className={DELETE_ACTION}
            >
              刪除
            </button>
          </>
        )}
      </div>

      {seasons.map((s) => (
        <SeasonLine
          key={s.id}
          term={s}
          fallbackZh={term.termZh}
          onEdit={onEdit}
          onDelete={onDelete}
          busy={busy}
        />
      ))}

      <DeleteTermDialog
        term={term}
        open={confirmDeleteOpen}
        onOpenChange={setConfirmDeleteOpen}
        onDelete={onDelete}
        description={
          seasonNote
            ? `確定要刪除「${term.termSrc} → ${term.termZh}」嗎？${seasonNote}也會一起刪掉。此操作無法復原。`
            : `確定要刪除「${term.termSrc} → ${term.termZh}」嗎？此操作無法復原。`
        }
      />
    </div>
  );
}

export interface GlossaryOrphanCardV2Props {
  termSrc: string;
  /** Season drawers whose show-wide row is gone. */
  seasons: GlossaryTerm[];
  onEdit: (termId: string, termZh: string) => Promise<void>;
  onDelete: (termId: string) => void;
  busy?: boolean;
}

/**
 * A term with season spellings but no show-wide row any more (legacy data:
 * deleting a card now deletes its seasons too). The main line names the term
 * only; its season lines can still be edited or deleted.
 */
export function GlossaryOrphanCardV2({
  termSrc,
  seasons,
  onEdit,
  onDelete,
  busy = false,
}: GlossaryOrphanCardV2Props) {
  return (
    <div
      data-testid={`glossary-orphan-${termSrc.toLowerCase()}`}
      className={`flex flex-col ${CARD_SHAPE}`}
    >
      <div className="flex min-h-[44px] items-center gap-3">
        <span className="truncate font-mono text-sm text-[var(--text-primary)]">{termSrc}</span>
        <span className="text-sm text-[var(--text-muted)]">全劇沒有這個詞</span>
      </div>
      {seasons.map((s) => (
        <SeasonLine key={s.id} term={s} onEdit={onEdit} onDelete={onDelete} busy={busy} />
      ))}
    </div>
  );
}
