// Design ref: ux-design.pen Screen B13p-D 修改資訊 · 類型 (Dcf86)
/**
 * GenreSelector — the 類型 field of the 修改資訊 dialog (poster-upload-a AC #3).
 *
 * Picked genres are chips with a ×; 「＋ 類型」 opens a menu of the rest. Values
 * are the zh-TW names the library stores (lib/genres `genreNamesFor`), and a
 * picked genre that is NOT in the menu (a Douban import, say) still shows as a
 * chip — a save must never drop a genre just because this list does not know it.
 */

import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import { Plus, X } from 'lucide-react';
import { cn } from '../../lib/utils';

export interface GenreSelectorProps {
  selected: string[];
  /** The genres the 「＋ 類型」 menu offers. */
  options: string[];
  onChange: (next: string[]) => void;
  /** id of the visible field label — the chip group is named by it. */
  labelId: string;
}

const MENU_ITEM =
  'flex min-h-11 cursor-default select-none items-center rounded-[var(--radius-sm)] px-3 text-sm text-[var(--text-primary)] outline-none data-[highlighted]:bg-[var(--accent-subtle)]';

export function GenreSelector({ selected, options, onChange, labelId }: GenreSelectorProps) {
  const remaining = options.filter((g) => !selected.includes(g));

  return (
    <div role="group" aria-labelledby={labelId} className="flex flex-wrap items-center gap-2">
      {selected.map((genre) => (
        <span
          key={genre}
          className="inline-flex h-7 items-center gap-0.5 rounded-full bg-[var(--accent-tint)] pl-3 pr-0.5 text-xs text-[var(--accent-text)]"
        >
          {genre}
          <button
            type="button"
            onClick={() => onChange(selected.filter((g) => g !== genre))}
            aria-label={`移除類型：${genre}`}
            className="flex h-6 w-6 items-center justify-center rounded-full transition-colors hover:bg-[var(--accent-subtle)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
          >
            <X className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        </span>
      ))}
      {remaining.length > 0 && (
        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild>
            <button
              type="button"
              className="inline-flex h-7 items-center gap-1 rounded-full border border-[var(--border-subtle)] px-3 text-xs text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
            >
              <Plus className="h-3.5 w-3.5" aria-hidden="true" />
              類型
            </button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content
              align="start"
              sideOffset={4}
              className="z-50 max-h-72 min-w-40 overflow-y-auto rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] p-1.5 shadow-[var(--shadow-lg)]"
            >
              {remaining.map((genre) => (
                <DropdownMenu.Item
                  key={genre}
                  onSelect={() => onChange([...selected, genre])}
                  className={cn(MENU_ITEM)}
                >
                  {genre}
                </DropdownMenu.Item>
              ))}
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      )}
    </div>
  );
}

export default GenreSelector;
