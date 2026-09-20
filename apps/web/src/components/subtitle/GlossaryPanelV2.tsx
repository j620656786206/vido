// Design ref: ux-design.pen Screen F6-D-v2 (dlfMR) + Screen F7-D-v2 (A85GFD) + Screen F6-SPEC-STATES (n3vIR) + Screen F6-M-v2 (buepS)
/**
 * Glossary management + review panel (ux3-subtitle-v2 AC 4, screens F6-D-v2
 * dlfMR / F6-M-v2 buepS / F7-D-v2 A85GFD 空狀態, states F6-SPEC-STATES n3vIR).
 * List / add / edit / per-row confirm / delete / 全部確認 batch-confirm
 * (party-mode P1) over the six 9R-15 routes. Four-state coverage (AC 5):
 * loading skeleton, empty (尚無詞彙 — 生成字幕時自動累積, distinct from
 * failure), fail-soft error + 重試, default.
 *
 * dsr-6c: a failed write says so (glossary-write-error), Esc inside the add
 * form or a row being edited cancels that instead of closing the panel, an
 * Enter or Esc that belongs to an input method is ignored, and adding a term
 * that is already in the table is stopped before the backend's upsert would
 * silently overwrite it.
 * Rule 5: list = query, writes = mutations (useGlossary.ts).
 */
import { useEffect, useRef, useState, type KeyboardEvent } from 'react';
import { BookOpen, CircleAlert } from 'lucide-react';
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '../ui/Dialog';
import { MOBILE_SHEET_CLOSE, MOBILE_SHEET_CONTENT, SheetGrabber } from '../ui/mobileSheet';
import { cn } from '../../lib/utils';
import { useGlossaryTerms, useGlossaryMutations } from '../../hooks/useGlossary';
import { isImeComposing } from '../../utils/keyboard';
import { GlossaryRowV2 } from './GlossaryRowV2';

export interface GlossaryPanelV2Props {
  /** STRING local media id (9R-15 route contract). */
  mediaId: string;
  mediaTitle: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * The add form sends no `language`, so the backend stores the default
 * (`models/glossary.go` GlossaryDefaultLanguage). The unique key is
 * (scope, term_src COLLATE NOCASE, language).
 */
const ADD_LANGUAGE = 'zh-Hant';

/** SQLite NOCASE folds ASCII A–Z only — `É` and `é` stay different rows. */
const foldAsciiCase = (s: string) => s.replace(/[A-Z]/g, (c) => c.toLowerCase());

/** Marks the add form and a row being edited; Esc inside one must not close the panel. */
const INLINE_EDITOR = '[data-glossary-inline-editor]';

const SECONDARY_BUTTON =
  'flex min-h-[44px] items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] disabled:opacity-50';

const TEXT_INPUT =
  'w-40 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none';

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
  const [writeError, setWriteError] = useState<string | null>(null);
  // The blocked term is kept by id and read from the live list, so the note
  // never quotes a translation that was edited or a row that was deleted.
  // `attempt` remounts the note so a second blocked submit is announced again.
  const [duplicate, setDuplicate] = useState<{ termId: string; attempt: number } | null>(null);

  // Every opening is a new session: it starts clean, and a write that was
  // still in flight from an earlier session cannot report into this one.
  // Reset on OPEN, during render (not in an effect), so the stale form never
  // paints and the closing panel does not collapse mid-animation. The parent
  // can also close the panel without going through onOpenChange.
  const [wasOpen, setWasOpen] = useState(open);
  const [session, setSession] = useState(0);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      setAdding(false);
      setDraftSrc('');
      setDraftZh('');
      setWriteError(null);
      setDuplicate(null);
      setSession((n) => n + 1);
    }
  }
  const sessionRef = useRef(session);
  useEffect(() => {
    sessionRef.current = session;
  }, [session]);

  const list = terms.data ?? [];
  const unconfirmedCount = list.filter((t) => !t.confirmed).length;
  const busy =
    add.isPending ||
    edit.isPending ||
    confirm.isPending ||
    confirmAll.isPending ||
    remove.isPending;

  /** Clears the last failure when a write starts; reports this one if it fails. */
  const runWrite = async <T,>(failureText: string, write: () => Promise<T>): Promise<T> => {
    const startedIn = sessionRef.current;
    setWriteError(null);
    try {
      return await write();
    } catch (err) {
      if (sessionRef.current === startedIn) setWriteError(failureText);
      throw err;
    }
  };
  const termSrcOf = (termId: string) => list.find((t) => t.id === termId)?.termSrc ?? '';

  const closeAddForm = () => {
    setAdding(false);
    setDraftSrc('');
    setDraftZh('');
    setDuplicate(null);
  };
  // While the add is on its way, 取消 and Esc wait: cancelling would throw
  // away the drafts that a failed add must hand back (dsr-6c CR MED-1).
  const cancelAddForm = () => {
    if (add.isPending) return;
    closeAddForm();
  };

  const submitAdd = () => {
    if (busy) return;
    const termSrc = draftSrc.trim();
    const termZh = draftZh.trim();
    if (!termSrc || !termZh) return;
    // The backend upserts: adding an existing term would overwrite its
    // translation, source and confirmed flag without a word (dsr-6c AC #8).
    const existing = terms.data?.find(
      (t) => t.language === ADD_LANGUAGE && foldAsciiCase(t.termSrc) === foldAsciiCase(termSrc)
    );
    if (existing) {
      setDuplicate((prev) => ({ termId: existing.id, attempt: (prev?.attempt ?? 0) + 1 }));
      return;
    }
    setDuplicate(null);
    // UI-added terms are manual + confirmed (the user just typed the mapping).
    runWrite(`新增「${termSrc}」失敗，請再試一次`, () =>
      add.mutateAsync({ termSrc, termZh, source: 'manual', confirmed: true })
    ).then(closeAddForm, () => {
      // Reported in glossary-write-error; the form keeps its drafts.
    });
  };

  const onAddFormKeyDown = (e: KeyboardEvent<HTMLElement>) => {
    if (isImeComposing(e)) return;
    if (e.key === 'Escape') cancelAddForm();
  };

  const duplicateTerm = duplicate && terms.data?.find((t) => t.id === duplicate.termId);

  const addForm = adding && (
    // order-[9998] keeps the form directly ABOVE the action pair (order-last =
    // 9999) on a phone, so it opens where 新增詞彙 was tapped rather than at the
    // top of a scrolled list. Inert on a desktop, where the body is not the
    // ordering context (the toolbar is max-sm:contents only).
    <div className="flex flex-col gap-2 max-sm:order-[9998]">
      <div
        data-testid="glossary-add-form"
        data-glossary-inline-editor=""
        className="flex items-center gap-2 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3.5 py-2 max-sm:flex-col max-sm:items-stretch max-sm:gap-2"
      >
        <input
          type="text"
          value={draftSrc}
          onChange={(e) => {
            setDraftSrc(e.target.value);
            setDuplicate(null);
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !isImeComposing(e)) submitAdd();
            else onAddFormKeyDown(e);
          }}
          placeholder="原文（例：Demogorgon）"
          aria-label="原文詞彙"
          data-testid="glossary-add-src"
          /* eslint-disable-next-line jsx-a11y/no-autofocus -- the form opens on the user's own 新增詞彙 click; focus follows the action */
          autoFocus
          className={`${TEXT_INPUT} max-sm:w-full font-mono`}
        />
        <input
          type="text"
          value={draftZh}
          onChange={(e) => setDraftZh(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !isImeComposing(e)) submitAdd();
            else onAddFormKeyDown(e);
          }}
          placeholder="譯名（例：魔王獸）"
          aria-label="中文譯名"
          data-testid="glossary-add-zh"
          className={`${TEXT_INPUT} max-sm:w-full`}
        />
        <span className="flex-1 max-sm:hidden" />
        {/* sm:contents drops this wrapper on a desktop — both buttons stay direct
            children of the form row. On a phone they share their own line. */}
        <div
          data-testid="glossary-add-actions"
          className="flex items-center gap-2 max-sm:justify-end sm:contents"
        >
          <button
            type="button"
            onClick={submitAdd}
            onKeyDown={onAddFormKeyDown}
            disabled={!draftSrc.trim() || !draftZh.trim()}
            // aria-disabled, not disabled, while a write runs: a button that turns
            // disabled under focus drops focus to <body>, and an Esc from there
            // would close the whole panel mid-save.
            aria-disabled={busy}
            data-testid="glossary-add-submit"
            className="flex min-h-[44px] items-center px-2.5 text-sm font-semibold text-[var(--accent-text)] disabled:opacity-50 aria-disabled:opacity-50"
          >
            新增
          </button>
          <button
            type="button"
            onClick={cancelAddForm}
            onKeyDown={onAddFormKeyDown}
            aria-disabled={add.isPending}
            data-testid="glossary-add-cancel"
            className="flex min-h-[44px] items-center px-2.5 text-sm text-[var(--text-secondary)] aria-disabled:opacity-50"
          >
            取消
          </button>
        </div>
      </div>
      {duplicate && duplicateTerm && (
        <p
          key={duplicate.attempt}
          role="alert"
          data-testid="glossary-add-duplicate"
          className="text-sm text-[var(--warning-text)]"
        >
          「{duplicateTerm.termSrc}」已經在表裡了（→ {duplicateTerm.termZh}
          ）。要改譯名，請按那一列的「編輯」。
        </p>
      )}
    </div>
  );

  const loadError = (
    <div
      data-testid="glossary-error"
      className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3"
    >
      <CircleAlert className="h-4 w-4 shrink-0 text-[var(--error-text)]" aria-hidden="true" />
      <p className="flex-1 text-sm text-[var(--error-text)]">名詞對照表載入失敗</p>
      <button
        type="button"
        onClick={() => terms.refetch()}
        data-testid="glossary-retry"
        className="flex min-h-[44px] shrink-0 items-center px-3 text-sm font-semibold text-[var(--accent-text)]"
      >
        重試
      </button>
    </div>
  );

  const listOrEmpty =
    list.length === 0 ? (
      !adding && (
        <div data-testid="glossary-empty" className="flex flex-col items-center gap-3 py-12">
          <BookOpen className="h-10 w-10 text-[var(--text-muted)]" aria-hidden="true" />
          <p className="text-base font-semibold text-[var(--text-primary)]">尚無詞彙</p>
          <p className="text-sm text-[var(--text-secondary)]">生成字幕時自動累積</p>
          <button
            type="button"
            onClick={() => setAdding(true)}
            data-testid="glossary-empty-add"
            className={`mt-2 ${SECONDARY_BUTTON}`}
          >
            新增詞彙
          </button>
        </div>
      )
    ) : (
      <div className="flex flex-col gap-2" data-testid="glossary-list">
        {list.map((term) => (
          <GlossaryRowV2
            key={term.id}
            term={term}
            busy={busy}
            onConfirm={(termId) => {
              runWrite(`確認「${termSrcOf(termId)}」失敗，請再試一次`, () =>
                confirm.mutateAsync(termId)
              ).catch(() => undefined);
            }}
            onEdit={(termId, termZh) =>
              runWrite(`「${termSrcOf(termId)}」的新譯名沒有存到，請再試一次`, () =>
                edit.mutateAsync({
                  termId,
                  termZh,
                  confirmed: list.find((t) => t.id === termId)?.confirmed ?? false,
                })
              )
            }
            onDelete={(termId) => {
              runWrite(`刪除「${termSrcOf(termId)}」失敗，請再試一次`, () =>
                remove.mutateAsync(termId)
              ).catch(() => undefined);
            }}
          />
        ))}
      </div>
    );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        data-testid="glossary-panel-v2"
        onEscapeKeyDown={(e) => {
          // Esc inside the add form or a row being edited cancels that (the
          // editor handles it); only an Esc from anywhere else closes the panel.
          if (e.target instanceof Element && e.target.closest(INLINE_EDITOR)) e.preventDefault();
        }}
        closeClassName={cn(MOBILE_SHEET_CLOSE, 'max-sm:top-4')}
        // A phone gets the shared bottom sheet (dsr-6f-1); every desktop size,
        // position and radius is sm:-prefixed, or MOBILE_SHEET_CONTENT's own
        // unprefixed w-full / max-w-none / translate-0 / rounded-b-none would
        // fight it in twMerge and one of the two breakpoints would lose.
        className={cn(
          'flex max-h-[85vh] flex-col gap-0 p-0 max-sm:overflow-hidden',
          MOBILE_SHEET_CONTENT,
          'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-[calc(100vw-2rem)] sm:max-w-[880px] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)] sm:border sm:border-[var(--border-subtle)]'
        )}
      >
        <SheetGrabber data-testid="glossary-sheet-grabber" />

        {/* Title bar — 44 high and borderless on a phone (F6-M-v2 buepS). */}
        <div
          data-testid="glossary-title-bar"
          className="flex h-14 items-center justify-between border-b border-[var(--border-subtle)] pl-6 pr-12 max-sm:h-11 max-sm:border-b-0 max-sm:pl-4"
        >
          <DialogTitle className="text-base font-semibold">名詞對照表 — {mediaTitle}</DialogTitle>
        </div>

        <div
          data-testid="glossary-body"
          className={cn(
            'flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-6 py-5 max-sm:gap-3.5 max-sm:px-4 max-sm:pt-1.5',
            // The bottom gutter is written once: the footer count carries the
            // safe area when it is there, the body when it is not.
            list.length > 0 ? 'max-sm:pb-4' : 'max-sm:pb-[max(1rem,env(safe-area-inset-bottom))]'
          )}
        >
          {/* On a phone this row dissolves (max-sm:contents) so the explainer and
              the action pair become body items — that is what lets the actions
              carry order-last down below the list, per buepS. */}
          <div
            data-testid="glossary-header-row"
            className="flex flex-col gap-3 max-sm:contents sm:flex-row sm:items-center"
          >
            <DialogDescription className="text-sm">生成字幕時會依此表固定譯名</DialogDescription>
            <span className="hidden flex-1 sm:block" />
            {/* sm:contents dissolves this wrapper on a desktop, so both buttons
                stay direct children of the toolbar row exactly as before. */}
            {(list.length > 0 || (terms.isError && !terms.data)) && (
              <div
                data-testid="glossary-actions"
                className="flex gap-3 max-sm:order-last max-sm:w-full max-sm:flex-col sm:contents"
              >
                {list.length > 0 && (
                  <button
                    type="button"
                    onClick={() => {
                      runWrite('全部確認失敗，請再試一次', () => confirmAll.mutateAsync()).catch(
                        () => undefined
                      );
                    }}
                    disabled={busy || unconfirmedCount === 0}
                    data-testid="glossary-confirm-all"
                    className={`${SECONDARY_BUTTON} max-sm:w-full`}
                  >
                    全部確認
                  </button>
                )}
                {/* On an empty list the empty state carries the only 新增詞彙. */}
                {(list.length > 0 || (terms.isError && !terms.data)) && (
                  <button
                    type="button"
                    onClick={() => setAdding(true)}
                    disabled={busy}
                    data-testid="glossary-add-term"
                    className={`${SECONDARY_BUTTON} max-sm:w-full`}
                  >
                    新增詞彙
                  </button>
                )}
              </div>
            )}
          </div>

          {/* Sticky, so a failure on a row far down the scrolling list is still
              in view — without scrolling that row (and its focused input) away.
              The opaque wrapper keeps rows from showing through the tint. */}
          {writeError && (
            <div
              data-testid="glossary-write-error-dock"
              className="sticky top-0 z-10 rounded-[var(--radius-md)] bg-[var(--bg-secondary)]"
            >
              <div
                role="alert"
                data-testid="glossary-write-error"
                className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3"
              >
                <CircleAlert
                  className="h-4 w-4 shrink-0 text-[var(--error-text)]"
                  aria-hidden="true"
                />
                <p className="flex-1 text-sm text-[var(--error-text)]">{writeError}</p>
              </div>
            </div>
          )}

          {addForm}

          {/* Four states: loading / error / empty / list. A failed background
              refetch keeps the list it already has and reports above it. */}
          {terms.isLoading ? (
            <SkeletonRows />
          ) : terms.isError && !terms.data ? (
            loadError
          ) : (
            <>
              {terms.isError && loadError}
              {listOrEmpty}
            </>
          )}
        </div>

        {/* Footer count — numbers Mono, zh Noto (DL-v2 font split). */}
        {list.length > 0 && (
          <div
            data-testid="glossary-footer-count"
            className="border-t border-[var(--border-subtle)] px-6 py-3.5 text-sm text-[var(--text-secondary)] max-sm:px-4 max-sm:pb-[max(0.875rem,env(safe-area-inset-bottom))]"
          >
            共 <span className="font-mono tabular-nums">{list.length}</span> 條 ·{' '}
            <span className="font-mono tabular-nums">{unconfirmedCount}</span> 條未確認
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
