# Web Application Specific Requirements

## Project-Type Overview

**Architecture Pattern: Single Page Application (SPA)**

Vido adopts modern SPA architecture:
- **Frontend**: React 19 + TanStack Router + TanStack Query
- **Backend**: Go/Gin RESTful API
- **Communication**: JSON over HTTP/HTTPS
- **State Management**: TanStack Query for server state, React hooks for UI state

**SPA Benefits for Vido:**
- Smooth user experience (no full page refreshes)
- Suitable for real-time updates (download progress, library changes)
- Offline caching strategy (PWA potential)
- Reduced server burden (static files + API)

---

## Browser Support Matrix

**Supported Browsers (1.0):**

| Browser | Minimum Version | Test Priority |
|---------|----------------|---------------|
| **Chrome** | Latest | P0 (Primary testing) |
| **Firefox** | Latest | P0 (Primary testing) |
| **Safari** | Latest | P1 (macOS NAS users) |
| **Edge** | Latest | P1 (Windows NAS users) |
| **iOS Safari** | iOS 15+ | P0 (Mobile access) |
| **Android Chrome** | Latest | P0 (Mobile access) |

**Explicitly NOT Supported:**
- ❌ Internet Explorer (any version)
- ❌ Legacy browsers (>2 years old)

**Browser Feature Requirements:**
- ES6+ JavaScript support
- CSS Grid & Flexbox
- Fetch API
- LocalStorage
- Intersection Observer (lazy loading)
- Modern CSS (CSS Variables, calc())

**Polyfills Strategy:**
- No polyfills provided (keep bundle lightweight)
- Unsupported browsers show upgrade prompt

---

## Responsive Design

**Breakpoints:**

```css
/* Mobile First Approach */
- Mobile: 320px - 767px (iPhone SE → iPhone 14 Pro Max)
- Tablet: 768px - 1023px (iPad)
- Desktop: 1024px+ (Desktop browsers, NAS admin interface)
```

**Layout Adaptations:**

**Mobile (320-767px):**
- Single column layout
- Grid view: 2 column posters
- Hide secondary information
- Bottom navigation bar
- Touch optimized (buttons >44px)

**Tablet (768-1023px):**
- Two column layout (sidebar + main content)
- Grid view: 3-4 column posters
- Show some secondary information

**Desktop (1024px+):**
- Three column layout (sidebar + main content + detail panel)
- Grid view: 4-6 column posters
- Full information display
- Mouse hover effects

**Touch vs Mouse Optimization:**
- Mobile devices: Large touch targets, swipe gestures, pull-to-refresh
- Desktop: Mouse hover, right-click menus, keyboard shortcuts

---

## UI Component Interaction Specifications

Defines the interactive behavior of contextual menus and quick-settings across the Media Library UI. Each item is scoped to its target epic/phase.

**1. Settings Gear Dropdown (Top Toolbar)**

The toolbar gear icon provides **library display preferences** (quick access). The full Settings page (Tab 4) covers system-level configuration (Epic 6).

| # | Item | Description | Scope |
|---|------|-------------|-------|
| 1 | Poster Size / Density | Small / Medium / Large (adjusts grid columns) | 1.0 - Epic 5 |
| 2 | Default Sort Preference | Remember preferred sort order | 1.0 - Epic 5 |
| 3 | Title Display Language | zh-TW priority / Original title priority | 1.0 - Epic 5 |
| 4 | Rescan Media Library | Manually trigger full library rescan | Growth |
| 5 | Cache Management | View/clear cache (FR53) | 1.0 - Epic 6 |

**2. PosterCard Context Menu (Hover `...` icon, top-right)**

Single media item quick actions accessible from the grid/list card.

| # | Item | Description | Scope |
|---|------|-------------|-------|
| 1 | View Details | Open Detail Panel (Story 5.6) | 1.0 - Epic 5 |
| 2 | Re-parse Metadata | Re-parse this item's metadata (FR40) | 1.0 - Epic 5 |
| 3 | Export Metadata | Export this item's metadata (FR40) | 1.0 - Epic 5 |
| 4 | Delete | Remove from library, requires confirmation dialog (FR40) | 1.0 - Epic 5 |
| — | *(separator)* | | |
| 5 | Mark Watched / Unwatched | Toggle watch status (FR45) | Growth - Epic 11 |
| 6 | Add to Collection | Add to custom collection (FR46) | Growth - Epic 11 |
| 7 | Edit Metadata | Manually modify metadata (FR21) | Growth |

**3. Detail Panel Context Menu (`...` icon, top-right next to close)**

Advanced operations when viewing full media details.

| # | Item | Description | Scope |
|---|------|-------------|-------|
| 1 | Re-parse Metadata | Re-parse metadata (FR40) | 1.0 - Epic 5 |
| 2 | Export Metadata | Export JSON/YAML/NFO (FR40) | 1.0 - Epic 5 |
| 3 | Delete | Remove from library, requires confirmation (FR40) | 1.0 - Epic 5 |
| — | *(separator)* | | |
| 4 | Edit Metadata | Manually modify all fields (FR21) | Growth |
| 5 | Mark Watched / Unwatched | Toggle watch status (FR45) | Growth - Epic 11 |
| 6 | Add to Collection | Add to custom collection (FR46) | Growth - Epic 11 |
| 7 | Find Subtitles | Search/download subtitles (FR75-80) | Growth - Epic 9 |
| 8 | Show in File System | Reveal actual file path | Growth |

**Interaction Rules:**
- Delete actions always appear last, use `--error` (red) color, and require a confirmation dialog
- Each menu item is prefixed with a Lucide icon (consistent with Design Brief icon rules)
- Items beyond current epic scope are **not rendered** until their epic is implemented
- PosterCard menu uses standard context menu (dropdown); Detail Panel menu uses the same pattern
- Mobile: Context menus trigger via long-press on cards; Detail Panel uses bottom sheet menu

---

## Performance Targets

**Page Load Performance:**

| Metric | Target | Measurement Condition |
|--------|--------|---------------------|
| **First Contentful Paint (FCP)** | <1.5s | First visit |
| **Largest Contentful Paint (LCP)** | <2.5s | First visit |
| **Time to Interactive (TTI)** | <3.5s | First visit |
| **Cumulative Layout Shift (CLS)** | <0.1 | Entire session |

**Runtime Performance:**

| Operation | Target Latency | Notes |
|-----------|---------------|-------|
| Page transition (routing) | <200ms | SPA navigation |
| Search response | <500ms | Local + API |
| Download status update | <5s | Polling interval |
| Grid scrolling | 60 FPS | Virtual scrolling |
| Image lazy load | <300ms | Intersection Observer |

**Bundle Size Targets:**

- Initial bundle: <500 KB (gzipped)
- Route-based code splitting
- Dynamic imports for heavy components
- Image optimization (WebP, responsive images)

---

## Real-Time Update Mechanism

**Strategy: Polling (1.0 Approach)**

**Why Polling for 1.0:**
- ✅ Simple implementation (TanStack Query built-in support)
- ✅ No WebSocket server maintenance needed
- ✅ Firewall/proxy friendly
- ✅ Automatic reconnection mechanism
- ⚠️ More resource-intensive (acceptable for self-hosted environment)

**Polling Configuration:**

```typescript
// qBittorrent download status
const { data: downloads } = useQuery({
  queryKey: ['downloads'],
  queryFn: fetchDownloads,
  refetchInterval: 5000, // 5 second polling
  refetchIntervalInBackground: false, // No polling in background
});

// Library updates (less frequent)
const { data: library } = useQuery({
  queryKey: ['library'],
  queryFn: fetchLibrary,
  refetchInterval: 30000, // 30 second polling
  staleTime: 10000, // Consider fresh for 10 seconds
});
```

**Smart Polling Optimization:**
- Stop polling when user not on current page (`refetchIntervalInBackground: false`)
- Immediately refresh when user returns (`refetchOnWindowFocus: true`)
- Auto-trigger library refresh after download completion
- Exponential backoff after errors (1s → 2s → 4s → 8s)

**Future Consideration (Post-1.0):**
- WebSocket or SSE for true real-time updates
- Reduce server load
- Better battery efficiency (mobile devices)

---

## SEO Strategy

**SEO Requirement: None**

Vido is a self-hosted tool deployed on private networks:
- ❌ No search engine indexing needed
- ❌ No Open Graph tags needed
- ❌ No sitemap.xml needed
- ❌ No SSR (Server-Side Rendering) needed

**Minimal Meta Tags (Good Practice):**
```html
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<title>Vido - Media Management</title>
```

---

## Accessibility

**1.0 Priority: Low (But Architecture Ready)**

**Minimum Requirements (1.0):**
- ✅ Semantic HTML (`<main>`, `<nav>`, `<article>`)
- ✅ Basic ARIA labels (`role`, `aria-label`)
- ✅ Keyboard navigation (Tab, Enter, Escape)
- ✅ Focus visibility (outline styles)
- ✅ Alt text (images, posters)

**Future Enhancements (Post-1.0):**
- WCAG 2.1 Level AA compliance
- Full screen reader support
- High contrast mode
- Font size adjustment
- Keyboard shortcut system

**Testing Approach:**
- Manual keyboard navigation testing
- Lighthouse Accessibility Score >70 (1.0 target)
- axe DevTools automated detection (during development)

---

## Frontend Tech Stack Detailed Specification

**Core Framework:**
- React 19 (latest stable)
- TypeScript (strict mode)
- Vite (build tool)

**Routing:**
- TanStack Router v1
- Type-safe routes
- Code splitting per route

**Data Fetching:**
- TanStack Query v5
- Optimistic updates
- Cache management

**Styling:**
- CSS Modules or Tailwind CSS (TBD)
- Responsive design utilities
- Dark mode support (architecture ready, 1.0 optional)

**Form Handling:**
- React Hook Form
- Zod schema validation

**State Management:**
- TanStack Query (server state)
- React Context + hooks (UI state)
- LocalStorage (user preferences)

**Testing:**
- Vitest (unit tests)
- React Testing Library (component tests)
- Playwright (E2E tests, post-1.0)

---

## Implementation Considerations

**Development Workflow:**
- Nx monorepo architecture (already established)
- Hot reload with Air (backend) and Vite HMR (frontend)
- OpenAPI/Swagger for API documentation
- Git-based version control

**Deployment:**
- Docker containerization
- Single Docker Compose deployment
- Frontend: Static files served by Nginx or backend
- Backend: Go binary with embedded static files option

**Progressive Enhancement:**
- Core functionality works without JavaScript (limited)
- Enhanced experience with JavaScript enabled
- Graceful degradation for unsupported browsers

**Internationalization (Future):**
- Architecture supports i18n (react-i18next ready)
- 1.0: Traditional Chinese + English only
- Post-1.0: Community translations
