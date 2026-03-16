# Story 6.0: Settings Page Foundation

Status: review

## Story

As a **Vido user**,
I want a **consistent settings page layout with organized navigation**,
So that **I can easily find and manage all system configuration options**.

## Design Reference

- **Desktop Settings Page:** `_bmad-output/screenshots/flow-c-search-filter-settings-desktop/10-settings-desktop.png`
- **Mobile Settings Page:** `_bmad-output/screenshots/flow-f-batch-settings-mobile/10-m-settings-mobile.png`
- Desktop gear dropdown (existing): `_bmad-output/screenshots/flow-c-search-filter-settings-desktop/01a-settings-gear-dropdown.png`
- Mobile quick settings (existing): `_bmad-output/screenshots/flow-f-batch-settings-mobile/01a-m-settings-bottom-sheet-mobile.png`
- **NOTE:** The gear dropdown and mobile bottom sheet are the EXISTING quick-settings (poster size, sort, language). These are SEPARATE from the full system settings page designed for this story. Dev must implement the full settings page per the Desktop/Mobile screenshots above.

## Acceptance Criteria

1. **AC1: Settings Page Shell with Sidebar Navigation**
   - Given the user navigates to `/settings`
   - When the settings page loads
   - Then a sidebar navigation displays all settings categories:
     - 連線設定 (Connection) — existing qBittorrent settings
     - 快取管理 (Cache)
     - 系統日誌 (Logs)
     - 服務狀態 (Status)
     - 備份與還原 (Backup & Restore)
     - 匯出/匯入 (Export/Import)
     - 效能監控 (Performance)
   - And each category has a Lucide icon (no emoji)
   - And the sidebar is on the left, content panel on the right

2. **AC2: Category Navigation and Deep-linking**
   - Given the settings sidebar is displayed
   - When clicking a category
   - Then the corresponding settings panel loads in the content area
   - And the active category is visually highlighted (blue accent)
   - And URL updates to `/settings/{category}` for deep-linking
   - And the routes are:
     - `/settings/connection` → 連線設定 (renders QBittorrentForm)
     - `/settings/cache` → 快取管理 (placeholder)
     - `/settings/logs` → 系統日誌 (placeholder)
     - `/settings/status` → 服務狀態 (placeholder)
     - `/settings/backup` → 備份與還原 (placeholder)
     - `/settings/export` → 匯出/匯入 (placeholder)
     - `/settings/performance` → 效能監控 (placeholder)

3. **AC3: Mobile Responsive Layout**
   - Given the settings page is displayed on mobile
   - When viewing on a small screen (<768px)
   - Then the sidebar collapses to horizontal scrollable tabs at the top
   - And the content panel takes full width below

4. **AC4: Default Route and Redirect**
   - Given the user navigates to `/settings` (no category)
   - When the page loads
   - Then it redirects to `/settings/connection` (first category)
   - And the 連線設定 category is highlighted as active

5. **AC5: Existing qBittorrent Integration**
   - Given the existing qBittorrent settings form exists
   - When the new settings shell is integrated
   - Then the `QBittorrentForm` component renders inside the `/settings/connection` content area
   - And all existing functionality (test connection, save config) works unchanged
   - And the old `/settings/qbittorrent` route redirects to `/settings/connection`

6. **AC6: Placeholder Content for Unimplemented Categories**
   - Given a category is not yet implemented (cache, logs, status, backup, export, performance)
   - When clicking that category
   - Then a "Coming Soon" placeholder is shown with:
     - Category icon and name
     - Brief description of what will be available
     - "此功能將在後續版本中提供" message

7. **AC7: TabNavigation Integration**
   - Given the global TabNavigation has a "設定" tab pointing to `/settings/qbittorrent`
   - When the settings foundation is complete
   - Then the "設定" tab `to` prop updates to `/settings` (redirects to `/settings/connection`)
   - And the `matchPaths: ['/settings']` continues to work for all settings sub-routes

## Tasks / Subtasks

- [x] Task 1: Create Settings Layout Route (AC: 1, 3)
  - [x] 1.1: Create `apps/web/src/routes/settings.tsx` as layout route with `<Outlet />`
  - [x] 1.2: Create `apps/web/src/components/settings/SettingsLayout.tsx` with sidebar + content area
  - [x] 1.3: Implement sidebar with 7 category items (Lucide icons + zh-TW labels)
  - [x] 1.4: Implement mobile responsive layout (sidebar → horizontal tabs below 768px)
  - [x] 1.5: Write tests for SettingsLayout component (12 tests)

- [x] Task 2: Setup Nested Routes (AC: 2, 4, 5)
  - [x] 2.1: Create `apps/web/src/routes/settings/index.tsx` — redirect to `/settings/connection`
  - [x] 2.2: Create `apps/web/src/routes/settings/connection.tsx` — renders QBittorrentForm
  - [x] 2.3: Create placeholder routes: `cache.tsx`, `logs.tsx`, `status.tsx`, `backup.tsx`, `export.tsx`, `performance.tsx`
  - [x] 2.4: Redirect old `apps/web/src/routes/settings/qbittorrent.tsx` to `/settings/connection`

- [x] Task 3: Create Placeholder Component (AC: 6)
  - [x] 3.1: Create `apps/web/src/components/settings/SettingsPlaceholder.tsx`
  - [x] 3.2: Accept `icon`, `title`, `description` props
  - [x] 3.3: Write tests for SettingsPlaceholder (5 tests)

- [x] Task 4: Update Global Navigation (AC: 7)
  - [x] 4.1: Update `TabNavigation.tsx` — change "設定" tab `to` from `/settings/qbittorrent` to `/settings`
  - [x] 4.2: Verify `matchPaths: ['/settings']` still highlights correctly for all sub-routes
  - [x] 4.3: Update TabNavigation tests — updated route paths

- [x] Task 5: Design Verification (AC: all)
  - [x] 5.1: UX Designer (Sally) verified desktop settings layout against design screenshot
  - [x] 5.2: UX Designer verified mobile responsive layout (sidebar → tabs with abbreviated labels)
  - [x] 5.3: Verified qBittorrent form works correctly in new shell
  - [x] 5.4: Verified all settings sub-routes load and URL deep-linking works
  - [x] 5.5: Verified redirect from `/settings` → `/settings/connection`
  - [x] 5.6: Verified redirect from `/settings/qbittorrent` → `/settings/connection`
  - [x] 5.7: 5 UX fixes applied: zh-TW labels, mobile tab abbreviations, tab border, button alignment, mobile button stacking

## Dev Notes

### Architecture: TanStack Router Nested Layout

The key pattern is creating a **layout route** at `routes/settings.tsx` that wraps all settings sub-routes:

```
routes/
├── settings.tsx              ← Layout route (SettingsLayout + <Outlet />)
└── settings/
    ├── index.tsx             ← Redirect to /settings/connection
    ├── connection.tsx        ← QBittorrentForm (migrated from qbittorrent.tsx)
    ├── cache.tsx             ← Placeholder
    ├── logs.tsx              ← Placeholder
    ├── status.tsx            ← Placeholder
    ├── backup.tsx            ← Placeholder
    ├── export.tsx            ← Placeholder
    └── performance.tsx       ← Placeholder
```

TanStack Router file-based routing: when `settings.tsx` exists alongside `settings/` directory, it becomes the layout parent. The layout component must render `<Outlet />` for child routes.

```typescript
// routes/settings.tsx
import { createFileRoute, Outlet } from '@tanstack/react-router';
import { SettingsLayout } from '../components/settings/SettingsLayout';

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
});

function SettingsPage() {
  return (
    <SettingsLayout>
      <Outlet />
    </SettingsLayout>
  );
}
```

### Existing Code to Reuse / Modify

| Component | Location | Action |
|-----------|----------|--------|
| `QBittorrentForm` | `components/settings/QBittorrentForm.tsx` | Reuse as-is inside `/settings/connection` |
| `ConnectionTestResult` | `components/settings/ConnectionTestResult.tsx` | No change needed |
| `TabNavigation` | `components/shell/TabNavigation.tsx` | Update settings tab `to` prop |
| `settings/qbittorrent.tsx` | `routes/settings/qbittorrent.tsx` | Delete or redirect to `/settings/connection` |

### Settings Categories Configuration

```typescript
// Suggested category config (components/settings/SettingsLayout.tsx)
import { Plug, Database, FileText, Activity, HardDrive, ArrowUpDown, Gauge } from 'lucide-react';

const SETTINGS_CATEGORIES = [
  { key: 'connection', label: '連線設定', icon: Plug, to: '/settings/connection' },
  { key: 'cache', label: '快取管理', icon: Database, to: '/settings/cache' },
  { key: 'logs', label: '系統日誌', icon: FileText, to: '/settings/logs' },
  { key: 'status', label: '服務狀態', icon: Activity, to: '/settings/status' },
  { key: 'backup', label: '備份與還原', icon: HardDrive, to: '/settings/backup' },
  { key: 'export', label: '匯出/匯入', icon: ArrowUpDown, to: '/settings/export' },
  { key: 'performance', label: '效能監控', icon: Gauge, to: '/settings/performance' },
] as const;
```

### Styling Patterns (from codebase)

**Desktop layout:**
```
┌─────────────────────────────────────────────┐
│ AppShell Header (sticky, z-50)              │
│ TabNavigation: 媒體庫 | 下載中 | 待解析 | 設定 │
├────────────┬────────────────────────────────┤
│ Sidebar    │ Content Area                   │
│ (w-56)     │ (<Outlet /> renders here)      │
│            │                                │
│ 🔌 連線設定 │ ┌──────────────────────────┐   │
│ 💾 快取管理 │ │ QBittorrentForm          │   │
│ 📄 系統日誌 │ │ or SettingsPlaceholder   │   │
│ ...        │ └──────────────────────────┘   │
└────────────┴────────────────────────────────┘
```

**Dark theme tokens (match existing):**
- Sidebar bg: `bg-slate-800/50` or `bg-slate-900`
- Sidebar item active: `bg-slate-700 text-blue-400 border-l-2 border-blue-400`
- Sidebar item inactive: `text-slate-400 hover:text-slate-200 hover:bg-slate-800`
- Content area: transparent (inherits `bg-slate-900` from root)
- Borders: `border-slate-700`

**Container:**
- Settings page: `mx-auto max-w-7xl px-4 py-6` (match library page width)
- Content panel: remove `max-w-2xl` from QBittorrentForm page wrapper (the sidebar + content layout handles width)

### Mobile Layout Pattern

Below `md` breakpoint (768px):
- Sidebar categories become horizontal scrollable tabs at top
- Pattern: `overflow-x-auto whitespace-nowrap flex gap-1`
- Active category: bottom border + blue text (similar to TabNavigation pattern)
- Content takes full width below

### Redirect Patterns

```typescript
// routes/settings/index.tsx — redirect to default category
import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/settings/')({
  beforeLoad: () => {
    throw redirect({ to: '/settings/connection' });
  },
});
```

For backward compat, either:
- Keep `settings/qbittorrent.tsx` with a redirect to `/settings/connection`
- Or delete it and rely on 404 → user navigates to new URL

### Pre-commit Verification

Before marking done:
- [ ] `routeTree.gen.ts` regenerated correctly with new nested routes
- [ ] All existing QBittorrent tests still pass (`QBittorrentForm.spec.tsx`, `ConnectionTestResult.spec.tsx`)
- [ ] TabNavigation tests updated for new settings link
- [ ] No console errors on any settings route
- [ ] Mobile responsive layout tested at 375px width

### Project Structure Notes

- All new components go in `apps/web/src/components/settings/`
- All new routes go in `apps/web/src/routes/settings/`
- Tests co-located: `SettingsLayout.spec.tsx`, `SettingsPlaceholder.spec.tsx`
- Follow existing patterns: `createFileRoute()`, Lucide icons, Tailwind dark theme

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-6-system-configuration-backup.md#Story 6.0]
- [Source: project-context.md] — All 16 mandatory rules
- [Source: apps/web/src/components/shell/AppShell.tsx] — Global shell pattern
- [Source: apps/web/src/components/shell/TabNavigation.tsx] — Tab active state matching
- [Source: apps/web/src/routes/settings/qbittorrent.tsx] — Existing settings route to migrate
- [Source: _bmad-output/implementation-artifacts/5-0-global-navigation-shell.md] — Story 5-0 pattern reference
- [Source: apps/web/src/routes/__root.tsx] — Root layout with AppShell wrapper
- [Source: _bmad-output/planning-artifacts/epic5-media-library-design-brief.md] — Design principles (no emoji, Lucide icons, dark theme)

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Tasks 1-4 implemented: SettingsLayout with sidebar/mobile tabs, 9 nested routes, SettingsPlaceholder, TabNavigation updated
- 91 test files, 1153 tests all passing (17 new tests: 12 SettingsLayout + 5 SettingsPlaceholder)
- qbittorrent.tsx preserved as redirect route for backward compatibility
- AppShell gear icon link updated from `/settings/qbittorrent` to `/settings`
- Task 5 (Design Verification) pending — requires SM/UX/User approval

### File List

#### Created Files
- `apps/web/src/routes/settings.tsx` — Layout route with SettingsLayout + Outlet
- `apps/web/src/routes/settings/index.tsx` — Redirect /settings/ → /settings/connection
- `apps/web/src/routes/settings/connection.tsx` — QBittorrentForm wrapper
- `apps/web/src/routes/settings/cache.tsx` — Placeholder
- `apps/web/src/routes/settings/logs.tsx` — Placeholder
- `apps/web/src/routes/settings/status.tsx` — Placeholder
- `apps/web/src/routes/settings/backup.tsx` — Placeholder
- `apps/web/src/routes/settings/export.tsx` — Placeholder
- `apps/web/src/routes/settings/performance.tsx` — Placeholder
- `apps/web/src/components/settings/SettingsLayout.tsx` — Sidebar + mobile tabs
- `apps/web/src/components/settings/SettingsLayout.spec.tsx` — 12 tests
- `apps/web/src/components/settings/SettingsPlaceholder.tsx` — Coming soon component
- `apps/web/src/components/settings/SettingsPlaceholder.spec.tsx` — 5 tests

#### Modified Files
- `apps/web/src/routes/settings/qbittorrent.tsx` — Changed to redirect to /settings/connection
- `apps/web/src/components/shell/TabNavigation.tsx` — Settings tab to="/settings"
- `apps/web/src/components/shell/TabNavigation.spec.tsx` — Updated test routes
- `apps/web/src/components/shell/AppShell.tsx` — Gear icon link to="/settings"
- `apps/web/src/components/shell/AppShell.spec.tsx` — Updated test routes
- `apps/web/src/routeTree.gen.ts` — Auto-regenerated with new route tree

## Changelog

| Change | Reason | Date |
|--------|--------|------|
| Story created | Settings page foundation needed for Epic 6 (Epic 5 retro finding) | 2026-03-16 |
| Tasks 1-4 implemented | SettingsLayout, nested routes, placeholders, nav update — 1153 tests passing | 2026-03-16 |
| Task 5 UX verification | Sally identified 5 discrepancies, all fixed: zh-TW labels, mobile tabs, button alignment — 1154 tests passing | 2026-03-16 |
