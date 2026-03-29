# UX Specification: Multi-Library Media Management

> **Author:** Sally (UX Designer)
> **Date:** 2026-03-29
> **Related PRD:** prd-multi-library-amendment.md
> **Status:** DRAFT

---

## 1. Setup Wizard — MediaFolderStep Redesign

### Current State
Single text input for one folder path.

### New Design: Multi-Library Setup

```
┌─────────────────────────────────────────────────────────┐
│  媒體庫設定                                               │
│  設定您的媒體資料夾路徑和類型。至少需要一個媒體庫。            │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ 📁 路徑  [/media/movies               ] [📂 瀏覽]  │ │
│  │ 🎬 類型  [電影 ▼]                                   │ │
│  │                                         [✕ 移除]   │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ 📁 路徑  [/media/tv                   ] [📂 瀏覽]  │ │
│  │ 🎬 類型  [影集 ▼]                                   │ │
│  │                                         [✕ 移除]   │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                           │
│  [+ 新增媒體庫]                                           │
│                                                           │
│  ┌──────────┐  ┌──────────────────────────────────────┐  │
│  │  上一步   │  │            下一步                     │  │
│  └──────────┘  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Interaction Details

**Adding a library entry:**
- Click "+ 新增媒體庫" adds a new row
- Default: empty path, type = 電影
- Path input: text field with placeholder `/media/movies`
- Type dropdown: 電影 (🎬) | 影集 (📺)
- Remove button: ✕ icon, shown only when >1 entry exists

**Validation:**
- At least one library entry required (disable "下一步" if empty)
- Path cannot be empty (show red border + error message)
- Duplicate paths show warning: "此路徑已被使用"
- Path validation on blur (API call to check if directory exists)

**Type dropdown options:**
| Value | Label | Icon |
|-------|-------|------|
| movie | 電影 | 🎬 |
| series | 影集 | 📺 |

---

## 2. Settings — Media Library Management

### Current State
Shows env var info with Docker setup guide. No edit capability.

### New Design: Library Management Cards

```
┌─────────────────────────────────────────────────────────┐
│  媒體庫管理                                               │
│  管理您的媒體資料夾和內容類型                                │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │  🎬 我的電影                            [⋮ 更多]    │ │
│  │  ─────────────────────────────────────────────────  │ │
│  │  📁 /media/movies                    ● 已連線       │ │
│  │  📁 /media/movies-4k                 ● 已連線       │ │
│  │                                                     │ │
│  │  2 個資料夾 · 32 部影片                              │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │  📺 電視影集                            [⋮ 更多]    │ │
│  │  ─────────────────────────────────────────────────  │ │
│  │  📁 /media/tv                        ● 已連線       │ │
│  │                                                     │ │
│  │  1 個資料夾 · 2,066 集                               │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                           │
│  [+ 新增媒體庫]                                           │
│                                                           │
│  ─────────────────────────────────────────────────────── │
│  掃描排程: [每天 ▼]                                       │
│  上次掃描: 2026-03-29 10:00 · 2,098 檔案 · 耗時 3m 42s   │
│  [🔍 掃描媒體庫]                                          │
└─────────────────────────────────────────────────────────┘
```

### Library Card Details

**Card header:**
- Type icon (🎬/📺) + Library name + overflow menu (⋮)
- Overflow menu: 編輯, 新增路徑, 刪除

**Card body:**
- List of paths with status indicator:
  - ● 綠色 = accessible (已連線)
  - ● 紅色 = not_found / not_readable (無法存取)
  - ● 灰色 = unknown (未檢查)

**Card footer:**
- Summary: folder count + media item count

### Edit Library Modal

```
┌─────────────────────────────────────────┐
│  編輯媒體庫                               │
│                                           │
│  名稱  [我的電影                  ]       │
│  類型  [電影 ▼]                           │
│                                           │
│  資料夾路徑:                               │
│  📁 /media/movies           [✕]          │
│  📁 /media/movies-4k        [✕]          │
│  [+ 新增路徑]                              │
│                                           │
│  ┌──────────┐  ┌────────────────────┐    │
│  │   取消    │  │      儲存變更       │    │
│  └──────────┘  └────────────────────┘    │
└─────────────────────────────────────────┘
```

### Delete Confirmation

```
┌─────────────────────────────────────────┐
│  ⚠️ 刪除媒體庫                           │
│                                           │
│  確定要刪除「我的電影」嗎？                  │
│                                           │
│  ☐ 同時移除已掃描的媒體資料                 │
│    (不勾選則保留媒體資料，僅移除庫設定)      │
│                                           │
│  ┌──────────┐  ┌────────────────────┐    │
│  │   取消    │  │      確認刪除       │    │
│  └──────────┘  └────────────────────┘    │
└─────────────────────────────────────────┘
```

---

## 3. Design Tokens

### Status Colors (consistent with existing design system)
- Accessible: `text-green-400` + `bg-green-900/30`
- Error: `text-red-400` + `bg-red-900/30`
- Unknown: `text-slate-400`

### Type Icons
- Movie: 🎬 or `<Film />` from lucide-react
- Series: 📺 or `<Tv />` from lucide-react

### Card Style (matching existing settings cards)
- Border: `border-slate-700`
- Background: `bg-slate-800`
- Padding: `p-6`
- Rounded: `rounded-lg`

---

## 4. Component Inventory

### New Components
| Component | Location | Description |
|-----------|----------|-------------|
| `MediaLibrarySetupStep` | `setup/MediaLibrarySetupStep.tsx` | Replaces `MediaFolderStep` |
| `MediaLibraryManager` | `settings/MediaLibraryManager.tsx` | Replaces media folder section in `ScannerSettings` |
| `LibraryCard` | `settings/LibraryCard.tsx` | Individual library display card |
| `LibraryEditModal` | `settings/LibraryEditModal.tsx` | Create/edit library modal |

### Modified Components
| Component | Change |
|-----------|--------|
| `SetupWizard.tsx` | Update step config to use new component |
| `ScannerSettings.tsx` | Replace folder display section with `MediaLibraryManager` |

### New Service
| Service | Description |
|---------|-------------|
| `mediaLibraryService.ts` | CRUD API calls for libraries and paths |

---

## 5. Responsive Considerations

- Library cards stack vertically on mobile
- Edit modal becomes full-screen sheet on mobile
- Path text truncates with ellipsis on narrow screens
- Type dropdown remains functional at all breakpoints

---

## 6. Accessibility

- All form inputs have associated labels
- Status indicators use both color AND text (not color-only)
- Delete confirmation requires explicit action (not swipe-to-delete)
- Keyboard navigation: Tab through cards, Enter to expand overflow menu
