// Design ref: ux-design.pen Screen AS-2 - Search Suggestions Dropdown (TMaw5)
// Source: ux-design.pen (Pencil app)
import { User } from 'lucide-react';
import type { Movie, Person, TVShow, UnifiedSearchResult } from '../../types/tmdb';
import { getImageUrl } from '../../lib/image';
import { cn } from '../../lib/utils';

// A keyboard-navigable suggestion (owned library items + movies + TV; people
// have no media detail page so they are displayed but not part of arrow-key
// navigation — AC #4). `id` is a TMDb numeric id for TMDb rows and the LOCAL
// string id for owned rows — `/media/$type/$id` branches on that shape
// (classifyId), so owned rows open the TMDb-independent local detail.
export interface NavigableItem {
  type: 'movie' | 'tv';
  id: number | string;
}

// searchOptionId returns the DOM id for the navigable option at a flat index
// (movies first, then TV). Shared by the listbox options (the `id` attribute)
// and the combobox input (`aria-activedescendant`) so screen readers announce
// the keyboard-highlighted row.
export function searchOptionId(index: number): string {
  return `search-option-${index}`;
}

// buildNavigableItems flattens the results into the order they are rendered
// (owned 媒體庫 items first, then TMDb movies, then TV). Shared by the parent
// (for arrow-key state) and exercised directly in tests.
export function buildNavigableItems(result?: UnifiedSearchResult): NavigableItem[] {
  if (!result) return [];
  const locals = [...(result.localMovies ?? []), ...(result.localTv ?? [])];
  return [
    ...locals.map((l): NavigableItem => ({ type: l.mediaType, id: l.id })),
    ...result.movies.map((m): NavigableItem => ({ type: 'movie', id: m.id })),
    ...result.tvShows.map((t): NavigableItem => ({ type: 'tv', id: t.id })),
  ];
}

// Maps a TMDb known_for_department to its zh-TW label, falling back to the raw
// English value for departments we have not localized.
const DEPARTMENT_ZH: Record<string, string> = {
  Directing: '導演',
  Acting: '演員',
  Writing: '編劇',
  Production: '製片',
  Sound: '音效',
  Camera: '攝影',
  Editing: '剪輯',
  Art: '美術',
};

function yearOf(date: string | undefined): string | null {
  if (!date) return null;
  const year = date.slice(0, 4);
  return /^\d{4}$/.test(year) ? year : null;
}

interface SearchSuggestionsProps {
  result?: UnifiedSearchResult;
  isLoading: boolean;
  query: string;
  /** Flat index (over movies+TV) of the keyboard-highlighted row, or -1. */
  activeIndex: number;
  onSelect: (item: NavigableItem) => void;
  onSubmitAll: () => void;
  onActiveIndexChange: (index: number) => void;
  listboxId?: string;
  /** When true (desktop), the dropdown floats below the input; when false
   * (mobile full-screen), it flows inline and fills the available height. */
  floating?: boolean;
}

export function SearchSuggestions({
  result,
  isLoading,
  query,
  activeIndex,
  onSelect,
  onSubmitAll,
  onActiveIndexChange,
  listboxId = 'search-suggestions',
  floating = true,
}: SearchSuggestionsProps) {
  const locals = [...(result?.localMovies ?? []), ...(result?.localTv ?? [])];
  const movies = result?.movies ?? [];
  const tvShows = result?.tvShows ?? [];
  const people = result?.people ?? [];
  const hasResults =
    locals.length > 0 || movies.length > 0 || tvShows.length > 0 || people.length > 0;

  return (
    <div
      className={cn(
        'overflow-hidden bg-[var(--bg-secondary)]',
        floating
          ? 'absolute left-0 right-0 top-full z-50 mt-2 rounded-xl border border-[var(--border-subtle)] shadow-xl'
          : 'mt-3 flex-1 overflow-y-auto'
      )}
      data-testid="search-suggestions"
    >
      {isLoading && (
        <div
          className="px-4 py-6 text-center text-sm text-[var(--text-muted)]"
          data-testid="search-suggestions-loading"
        >
          搜尋中…
        </div>
      )}

      {!isLoading && !hasResults && (
        <div
          className="px-4 py-6 text-center text-sm text-[var(--text-muted)]"
          data-testid="search-suggestions-empty"
        >
          找不到「{query}」的結果
        </div>
      )}

      {!isLoading && hasResults && (
        <>
          {/* Only the keyboard-navigable options (movies + TV) live inside the
              listbox so its descendants are valid `option`/`group` nodes that
              `aria-activedescendant` can reference. */}
          {(locals.length > 0 || movies.length > 0 || tvShows.length > 0) && (
            <div role="listbox" id={listboxId} aria-label="搜尋建議">
              {locals.length > 0 && (
                <Section label="媒體庫">
                  {locals.map((hit, i) => (
                    <MediaRow
                      key={`local-${hit.id}`}
                      id={searchOptionId(i)}
                      title={hit.title}
                      originalTitle={hit.originalTitle ?? ''}
                      year={yearOf(hit.releaseDate)}
                      rating={0}
                      posterPath={hit.posterPath ?? null}
                      owned
                      active={activeIndex === i}
                      onSelect={() => onSelect({ type: hit.mediaType, id: hit.id })}
                      onHover={() => onActiveIndexChange(i)}
                    />
                  ))}
                </Section>
              )}

              {movies.length > 0 && (
                <Section label="電影">
                  {movies.map((movie, i) => (
                    <MediaRow
                      key={`movie-${movie.id}`}
                      id={searchOptionId(locals.length + i)}
                      title={movie.title}
                      originalTitle={movie.originalTitle}
                      year={yearOf(movie.releaseDate)}
                      rating={movie.voteAverage}
                      posterPath={movie.posterPath}
                      active={activeIndex === locals.length + i}
                      onSelect={() => onSelect({ type: 'movie', id: movie.id })}
                      onHover={() => onActiveIndexChange(locals.length + i)}
                    />
                  ))}
                </Section>
              )}

              {tvShows.length > 0 && (
                <Section label="影集">
                  {tvShows.map((show, i) => (
                    <MediaRow
                      key={`tv-${show.id}`}
                      id={searchOptionId(locals.length + movies.length + i)}
                      title={show.name}
                      originalTitle={show.originalName}
                      year={yearOf(show.firstAirDate)}
                      rating={show.voteAverage}
                      posterPath={show.posterPath}
                      active={activeIndex === locals.length + movies.length + i}
                      onSelect={() => onSelect({ type: 'tv', id: show.id })}
                      onHover={() => onActiveIndexChange(locals.length + movies.length + i)}
                    />
                  ))}
                </Section>
              )}
            </div>
          )}

          {/* People are not keyboard-navigable (no person detail page) so they
              render outside the listbox as a plain, non-option section. */}
          {people.length > 0 && (
            <div className="py-2">
              <div className="px-4 py-1 text-xs font-medium tracking-wide text-[var(--text-muted)]">
                人物
              </div>
              {people.map((person) => (
                <PersonRow key={`person-${person.id}`} person={person} />
              ))}
            </div>
          )}

          <button
            type="button"
            onClick={onSubmitAll}
            className="flex w-full items-center justify-center gap-1 border-t border-[var(--border-subtle)] px-4 py-2 text-sm text-[var(--accent-primary)] transition-colors hover:bg-[var(--bg-tertiary)]"
            data-testid="search-suggestions-submit-all"
          >
            按 Enter 查看所有結果 →
          </button>
        </>
      )}
    </div>
  );
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div role="group" aria-label={label} className="py-2">
      <div
        aria-hidden="true"
        className="px-4 py-1 text-xs font-medium tracking-wide text-[var(--text-muted)]"
      >
        {label}
      </div>
      {children}
    </div>
  );
}

interface MediaRowProps {
  id: string;
  title: string;
  originalTitle: string;
  year: string | null;
  rating: number;
  posterPath: string | null;
  /** Owned 媒體庫 row — shows the 已擁有 badge instead of a TMDb rating. */
  owned?: boolean;
  active: boolean;
  onSelect: () => void;
  onHover: () => void;
}

function MediaRow({
  id,
  title,
  originalTitle,
  year,
  rating,
  posterPath,
  owned = false,
  active,
  onSelect,
  onHover,
}: MediaRowProps) {
  const poster = getImageUrl(posterPath, 'w92');
  const meta = [
    originalTitle && originalTitle !== title ? originalTitle : null,
    year && `(${year})`,
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <button
      type="button"
      id={id}
      role="option"
      aria-selected={active}
      onClick={onSelect}
      onMouseMove={onHover}
      className={cn(
        'flex w-full items-center gap-3 px-4 py-2 text-left transition-colors',
        active ? 'bg-[var(--bg-tertiary)]' : 'hover:bg-[var(--bg-tertiary)]'
      )}
      data-testid="search-suggestion-item"
    >
      <div className="h-14 w-10 shrink-0 overflow-hidden rounded bg-[var(--bg-tertiary)]">
        {poster && (
          <img src={poster} alt="" className="h-full w-full object-cover" loading="lazy" />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm text-white">{title}</div>
        <div className="truncate text-xs text-[var(--text-muted)]">
          {meta}
          {rating > 0 && (
            <span className="ml-1">
              {meta && '· '}★ {rating.toFixed(1)}
            </span>
          )}
        </div>
      </div>
      {owned && (
        <span
          className="shrink-0 rounded-full bg-[var(--accent-primary)]/15 px-2 py-0.5 text-xs text-[var(--accent-primary)]"
          data-testid="search-suggestion-owned-badge"
        >
          已擁有
        </span>
      )}
    </button>
  );
}

function PersonRow({ person }: { person: Person }) {
  const profile = getImageUrl(person.profilePath, 'w92');
  const department = DEPARTMENT_ZH[person.knownForDepartment] ?? person.knownForDepartment;
  const subtitle = [
    department,
    person.originalName && person.originalName !== person.name ? person.originalName : null,
  ]
    .filter(Boolean)
    .join(' · ');

  return (
    <div
      className="flex cursor-default items-center gap-3 px-4 py-2"
      data-testid="search-suggestion-person"
    >
      <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-[var(--bg-tertiary)] text-[var(--text-muted)]">
        {profile ? (
          <img src={profile} alt="" className="h-full w-full object-cover" loading="lazy" />
        ) : (
          <User className="h-5 w-5" />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm text-white">{person.name}</div>
        {subtitle && <div className="truncate text-xs text-[var(--text-muted)]">{subtitle}</div>}
      </div>
    </div>
  );
}

// Re-export the item types consumers may need.
export type { Movie, TVShow, Person };
