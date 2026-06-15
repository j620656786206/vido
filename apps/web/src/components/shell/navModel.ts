// Implements: <utility — no .pen counterpart>
/**
 * Shared navigation model for the v2 shell (UX Redesign Phase 2 — UX2-1).
 * One source of truth so the desktop sidebar, the collapsed rail, the mobile tab
 * bar, and the More sheet agree on destinations, icons, labels, and testids.
 *
 * SCOPE: only routes that exist today earn a slot. `活動 /activity` lands in Phase 3
 * (ux3-2-3, this change); `系統 /system` is still DEFERRED (route not built yet). The
 * `媒體庫` children point at the clean `/library/movies` · `/library/tv` routes.
 */
import {
  House,
  Library,
  Film,
  Tv,
  Compass,
  Activity,
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

// 任務 (tasks) destinations — 活動 + 下載 + 設定 (系統 still deferred, route not built).
export const ACTIVITY: NavDest = {
  key: 'activity',
  label: '活動',
  to: '/activity',
  icon: Activity,
};
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
export const RAIL_DESTS: NavDest[] = [HOME, LIBRARY, DISCOVER, ACTIVITY, DOWNLOADS, SETTINGS];

/**
 * Mobile bottom-4 (§6.3). ux3-2-3 lands the design's 4th slot — `活動` now goes live
 * (flow-k-activity-v2 A2-M-v2: 首頁 · 媒體庫 · 活動 · 下載). `探索` moves into the More
 * sheet; the 5th slot opens it.
 */
export const MOBILE_TABS: NavDest[] = [HOME, LIBRARY, ACTIVITY, DOWNLOADS];

/** Destinations that live in the mobile More sheet (everything off the bottom-4). */
export const MORE_DESTS: NavDest[] = [DISCOVER, SETTINGS];
