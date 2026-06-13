// Implements: <utility — no .pen counterpart>
/**
 * Shared navigation model for the v2 shell (UX Redesign Phase 2 — UX2-1).
 * One source of truth so the desktop sidebar, the collapsed rail, the mobile tab
 * bar, and the More sheet agree on destinations, icons, labels, and testids.
 *
 * PILOT SCOPE (ADR D2 + story AC #4): only routes that exist today earn a slot.
 * `活動 /activity` and `系統 /system` are DEFERRED — their routes do not exist yet
 * (Phase 3). The `媒體庫` children point at the working `?type=` deep links
 * (`/library/movies`,`/library/tv` are created in UX2-2; until then these links
 * stay valid and never 404 — UX2-2 repoints them once the child routes land).
 */
import {
  House,
  Library,
  Film,
  Tv,
  Compass,
  Download,
  Settings,
  type LucideIcon,
} from 'lucide-react';

export interface NavDest {
  /** testid suffix → `nav-{key}`. */
  key: string;
  label: string;
  /** Route path (a real, existing route). */
  to: string;
  /** Optional search params (e.g. the `?type=` library deep link). */
  search?: Record<string, string>;
  icon: LucideIcon;
  /** Exact active match (Home `/` must not light up on every route). */
  exact?: boolean;
}

// 內容 (content) destinations — always structurally above 任務 (N3).
export const HOME: NavDest = { key: 'home', label: '首頁', to: '/', icon: House, exact: true };
export const LIBRARY: NavDest = { key: 'library', label: '媒體庫', to: '/library', icon: Library };
export const MOVIES: NavDest = {
  key: 'movies',
  label: '電影',
  to: '/library',
  search: { type: 'movie' },
  icon: Film,
};
export const TV: NavDest = {
  key: 'tv',
  label: '影集',
  to: '/library',
  search: { type: 'tv' },
  icon: Tv,
};
export const DISCOVER: NavDest = { key: 'discover', label: '探索', to: '/discover', icon: Compass };

// 任務 (tasks) destinations — pilot subset (下載 + 設定; 活動/系統 deferred).
export const DOWNLOADS: NavDest = {
  key: 'downloads',
  label: '下載',
  to: '/downloads',
  icon: Download,
};
export const SETTINGS: NavDest = {
  key: 'settings',
  label: '設定',
  to: '/settings',
  icon: Settings,
};

/** Top-level destinations shown on the collapsed 64px rail (§6.2 budget). */
export const RAIL_DESTS: NavDest[] = [HOME, LIBRARY, DISCOVER, DOWNLOADS, SETTINGS];

/**
 * Mobile bottom-4 (§6.3). The design's 4th slot is `活動`, but /activity does not
 * exist in the pilot — `探索` (a real content destination) fills the 4th slot
 * until Phase 3 ships /activity. 5th slot opens the More sheet.
 */
export const MOBILE_TABS: NavDest[] = [HOME, LIBRARY, DISCOVER, DOWNLOADS];

/** Destinations that live in the mobile More sheet (everything off the bottom-4). */
export const MORE_DESTS: NavDest[] = [SETTINGS];
