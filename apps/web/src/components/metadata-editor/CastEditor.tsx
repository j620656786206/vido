// Design ref: ux-design.pen Screen B13p-D 修改資訊 · 演員 (r7iTr)
/**
 * CastEditor — the 演員 field of the 修改資訊 dialog (poster-upload-a AC #4).
 *
 * Each actor is a neutral chip with a ×; 「＋ 演員」 turns into a text box:
 * Enter or leaving the box adds, Esc cancels. Blank names and repeats are ignored.
 */

import { useEffect, useRef, useState } from 'react';
import { Plus, X } from 'lucide-react';

export interface CastEditorProps {
  cast: string[];
  onChange: (next: string[]) => void;
  /** id of the visible field label — the chip group is named by it. */
  labelId: string;
}

export function CastEditor({ cast, onChange, labelId }: CastEditorProps) {
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  // Esc unmounts the box, and the browser blurs it on the way out — that blur
  // must not re-add the name the user just cancelled.
  const cancelledRef = useRef(false);
  // The box replaces the button the user just pressed, so focus follows it.
  useEffect(() => {
    if (adding) {
      cancelledRef.current = false;
      inputRef.current?.focus();
    }
  }, [adding]);

  const add = () => {
    const name = draft.trim();
    if (name && !cast.includes(name)) onChange([...cast, name]);
    setDraft('');
  };

  return (
    <div role="group" aria-labelledby={labelId} className="flex flex-wrap items-center gap-2">
      {cast.map((name) => (
        <span
          key={name}
          className="inline-flex h-7 items-center gap-0.5 rounded-full bg-[var(--bg-tertiary)] pl-3 pr-0.5 text-xs text-[var(--text-secondary)]"
        >
          {name}
          <button
            type="button"
            onClick={() => onChange(cast.filter((c) => c !== name))}
            aria-label={`移除演員：${name}`}
            className="flex h-6 w-6 items-center justify-center rounded-full transition-colors hover:bg-[var(--bg-primary)] hover:text-[var(--text-primary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
          >
            <X className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        </span>
      ))}
      {adding ? (
        <input
          ref={inputRef}
          type="text"
          value={draft}
          aria-label="新增演員"
          // MetadataEditorDialog reads this so Esc cancels the box, not the dialog.
          data-escape-local="true"
          placeholder="輸入演員名稱後按 Enter"
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              // Enter inside a form would submit it.
              e.preventDefault();
              add();
            } else if (e.key === 'Escape') {
              e.preventDefault();
              cancelledRef.current = true;
              setDraft('');
              setAdding(false);
            }
          }}
          // Leaving the box (Tab, a click on 儲存) keeps what was typed — only
          // Esc cancels. Otherwise a name typed without Enter vanishes on save.
          onBlur={() => {
            if (!cancelledRef.current) add();
            setDraft('');
            setAdding(false);
          }}
          className="h-7 w-44 rounded-full border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 text-xs text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-transparent focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
        />
      ) : (
        <button
          type="button"
          onClick={() => setAdding(true)}
          className="inline-flex h-7 items-center gap-1 rounded-full border border-[var(--border-subtle)] px-3 text-xs text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
        >
          <Plus className="h-3.5 w-3.5" aria-hidden="true" />
          演員
        </button>
      )}
    </div>
  );
}

export default CastEditor;
