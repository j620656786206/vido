# Vido Design Guidelines — Developer Reference

> **Purpose:** Actionable reference for developers implementing UI. Maps UX spec Design System definitions to Tailwind config, CSS variables, and component patterns.
>
> **Source:** Extracted from `ux-design-specification.md` (sections: Design System Foundation, Visual Design Foundation, Component Strategy, Design System Integration Summary).
>
> **Last Updated:** 2026-04-04

---

## 1. Color Token Reference

### Background Colors (Dark Theme — Midnight Blue)

| CSS Variable | HSL Value | Tailwind Class | Usage |
|---|---|---|---|
| `--bg-primary` | `hsl(222, 47%, 11%)` | `bg-primary` | Main canvas, page background |
| `--bg-secondary` | `hsl(217, 33%, 17%)` | `bg-secondary` | Cards, panels, secondary surfaces |
| `--bg-tertiary` | `hsl(215, 28%, 23%)` | `bg-tertiary` | Hover states, tertiary surfaces |

### Brand & Accent Colors

| CSS Variable | HSL Value | Tailwind Class | Usage |
|---|---|---|---|
| `--accent-primary` | `hsl(217, 91%, 60%)` | `bg-accent` / `text-accent` | Primary actions, links, focus rings |
| `--accent-hover` | `hsl(217, 91%, 70%)` | `hover:bg-accent-hover` | Hover state for accent elements |
| `--accent-pressed` | `hsl(217, 91%, 50%)` | `active:bg-accent-pressed` | Active/pressed state |

### Semantic Colors

| CSS Variable | HSL Value | Tailwind Class | Usage | Never Use For |
|---|---|---|---|---|
| `--success` | `hsl(142, 76%, 36%)` | `bg-success` / `text-success` | Success, completion | Primary buttons |
| `--error` | `hsl(0, 84%, 60%)` | `bg-error` / `text-error` | Errors, destructive actions | Success, primary |
| `--warning` | `hsl(38, 92%, 50%)` | `bg-warning` / `text-warning` | Warnings, partial success | Errors, primary |
| `--info` | `hsl(200, 98%, 48%)` | `bg-info` / `text-info` | Informational messages | — |

### Text Colors

| CSS Variable | HSL Value | Tailwind Class | Usage | WCAG on bg-primary |
|---|---|---|---|---|
| `--text-primary` | `hsl(0, 0%, 95%)` | `text-foreground` | Headings, primary content | AAA (16.6:1) |
| `--text-secondary` | `hsl(0, 0%, 70%)` | `text-foreground-secondary` | Descriptions, metadata | AAA (7.8:1) |
| `--text-muted` | `hsl(0, 0%, 50%)` | `text-foreground-muted` | Timestamps, tertiary info | AA (4.6:1) |
| `--text-inverse` | `hsl(222, 47%, 11%)` | `text-foreground-inverse` | Dark text on light backgrounds | — |

### Color Blindness Rule

Success (green) and Error (red) must **never** be used as sole differentiators. Always pair with icons and text labels.

---

## 2. Typography

### Font Stacks

**Primary (body text):**
```
'Noto Sans TC', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif
```
- Tailwind class: `font-sans` (after config update)
- `font-display: swap` to prevent FOIT
- Subset to Traditional Chinese characters for performance

**Monospace (code, URLs, tech info):**
```
'JetBrains Mono', 'Consolas', 'Monaco', monospace
```
- Tailwind class: `font-mono` (after config update)

### Type Scale

| Level | Size | Line Height | Weight | Tailwind | Use Case |
|---|---|---|---|---|---|
| H1 | 2rem (32px) | 1.4 | 700 | `text-3xl font-bold` | Page titles |
| H2 | 1.5rem (24px) | 1.5 | 700 | `text-2xl font-bold` | Section headings |
| H3 | 1.25rem (20px) | 1.6 | 600 | `text-xl font-semibold` | Subsection headings |
| Body Large | 1.125rem (18px) | 1.6 | 500 | `text-lg font-medium` | Emphasis text |
| Body | 1rem (16px) | 1.6 | 400 | `text-base` | Default body text |
| Body Small | 0.875rem (14px) | 1.5 | 400 | `text-sm` | Metadata, secondary |
| Caption | 0.75rem (12px) | 1.5 | 400 | `text-xs` | Timestamps, labels |

### Traditional Chinese Rules

- **No letter-spacing** for zh-TW body text (characters are inherently well-spaced)
- Tighter letter-spacing (`-0.02em`) for Latin headings
- Wide letter-spacing (`0.05em`) for ALL-CAPS UI labels (STATUS, SOURCE)
- Line height 1.6 is optimal for zh-TW (taller x-height than Latin)

---

## 3. Spacing System

### 4px Modular Scale

All spacing follows a 4px base unit. Use standard Tailwind spacing utilities (`p-4`, `gap-6`, `m-8`).

### Component Spacing Rules

| Context | Size | Tailwind | Value |
|---|---|---|---|
| **Card padding** (small) | space-4 | `p-4` | 16px |
| **Card padding** (standard) | space-6 | `p-6` | 24px |
| **Card padding** (large) | space-8 | `p-8` | 32px |
| **Between sections** | space-12 to space-16 | `py-12` to `py-16` | 48-64px |
| **Within sections** | space-8 | `gap-8` | 32px |
| **Poster grid gap** | space-4 | `gap-4` | 16px |
| **Component grid gap** | space-6 | `gap-6` | 24px |
| **Form field gap** | space-4 | `gap-4` | 16px |

---

## 4. Border Radius & Shadows

### Border Radius Scale

| Token | Value | Tailwind Class | Usage |
|---|---|---|---|
| `--radius-sm` | 4px | `rounded-sm` | Badges, tags |
| `--radius-md` | 8px | `rounded-md` | Buttons, inputs, small cards |
| `--radius-lg` | 12px | `rounded-lg` | Standard cards, posters |
| `--radius-xl` | 16px | `rounded-xl` | Large modals, panels |
| `--radius-2xl` | 24px | `rounded-2xl` | Hero sections (rare) |
| `--radius-full` | 9999px | `rounded-full` | Pills, circular avatars |

### Shadow Elevation System

| Token | Value | Tailwind Class | Usage |
|---|---|---|---|
| `--shadow-sm` | `0 1px 2px rgba(0,0,0,0.3)` | `shadow-sm` | Buttons (subtle depth) |
| `--shadow-md` | `0 4px 8px rgba(0,0,0,0.4)` | `shadow-md` | Cards default |
| `--shadow-lg` | `0 8px 16px rgba(0,0,0,0.5)` | `shadow-lg` | Hover states, dropdowns, progress cards |
| `--shadow-xl` | `0 12px 24px rgba(0,0,0,0.6)` | `shadow-xl` | Modals, dialogs, poster hover |
| `--shadow-2xl` | `0 24px 48px rgba(0,0,0,0.7)` | `shadow-2xl` | Critical alerts, side panels |

**Standard pattern:** Cards use `shadow-md` default → `shadow-xl` on hover.

---

## 5. Animation & Transitions

### Durations

| Token | Value | Usage |
|---|---|---|
| `--duration-fast` | 150ms | Micro-interactions (hover, focus) |
| `--duration-base` | 300ms | Standard transitions (card appear, color change) |
| `--duration-slow` | 500ms | Complex animations (panel slide, modal open) |

### Easing

- **`ease-out`** for enter animations (most common)
- **`ease-in`** for exit animations
- **`ease-in-out`** for looping animations

### Standard Patterns

```css
/* Hover states */
transition: all 150ms ease-out;

/* Card appearance */
transition: opacity 300ms ease-out, transform 300ms ease-out;

/* Panel slide-in */
transition: transform 500ms ease-out;
```

### Hover Effects

- Poster cards: `hover:scale-105 hover:shadow-xl hover:-translate-y-1`
- Buttons: `transition-all duration-150`
- Cards: background tint via rgba overlay

### Accessibility

```css
@media (prefers-reduced-motion: reduce) {
  * { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; }
}
```

---

## 6. Responsive Breakpoints & Grid

### Breakpoints

| Name | Min Width | Tailwind Prefix | Grid Columns |
|---|---|---|---|
| Mobile | 320px | (default) | 4-column, 12px gutters |
| Tablet | 768px | `md:` | 8-column, 16px gutters |
| Desktop | 1024px | `lg:` | 12-column, 24px gutters |
| Wide | 1440px | `xl:` | 12-column, 24px gutters |

### Poster Grid Density

```css
/* Desktop (1024px+): 5-6 posters per row */
grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
gap: 16px; /* gap-4 */

/* Tablet (768-1023px): 3-4 posters per row */
grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
gap: 12px; /* gap-3 */

/* Mobile (<768px): 2 posters per row */
grid-template-columns: repeat(2, 1fr);
gap: 12px; /* gap-3 */
```

Tailwind shortcut (after config):
```html
<div class="grid grid-cols-media-grid gap-4 md:gap-3">
```

### Layout Density Principle

- **Desktop:** Dense but organized (maximize screen real estate)
- **Tablet:** Balanced (comfortable browsing)
- **Mobile:** Spacious (touch-friendly, min 44x44px touch targets)

---

## 7. Component Variant Pattern (CVA)

All custom components should use [class-variance-authority](https://cva.style/) for type-safe variants.

### Pattern

```tsx
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/utils/cn'; // cn = clsx + tailwind-merge

const componentVariants = cva(
  'base-classes-here', // shared base styles
  {
    variants: {
      variant: {
        primary: 'bg-accent text-white hover:bg-accent/90',
        secondary: 'border border-border hover:bg-secondary',
        ghost: 'hover:bg-tertiary',
        destructive: 'bg-error text-white hover:bg-error/90',
      },
      size: {
        sm: 'px-3 py-1.5 text-sm h-9',
        md: 'px-4 py-2 text-base h-11',
        lg: 'px-6 py-3 text-lg h-13',
      },
    },
    defaultVariants: {
      variant: 'primary',
      size: 'md',
    },
  }
);

interface ComponentProps extends VariantProps<typeof componentVariants> {
  className?: string;
}

export function Component({ variant, size, className, ...props }: ComponentProps) {
  return <div className={cn(componentVariants({ variant, size }), className)} {...props} />;
}
```

### `cn` Utility

Already available via `clsx` + `tailwind-merge` (both installed):

```ts
// utils/cn.ts
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

### Component Naming Conventions

| Type | Location | Naming | Example |
|---|---|---|---|
| Shared UI primitives | `components/ui/` | PascalCase | `Button.tsx`, `Badge.tsx` |
| Feature components | `components/{feature}/` | PascalCase | `PosterCard.tsx`, `HeroBanner.tsx` |
| CVA variant files | Same as component | Inline or co-located | Variants defined in same file |
| Test files | Co-located | `*.spec.tsx` | `Button.spec.tsx` |

---

## 8. Iconography

**Primary icon library:** `lucide-react` (already installed)

| Size | Value | Usage |
|---|---|---|
| Inline | 16px (`size={16}`) | Inline with text |
| Button | 24px (`size={24}`) | Inside buttons, nav items |
| Empty state | 48px (`size={48}`) | Empty state illustrations |

### Rules

- Always pair icons with text labels (except icon-only buttons)
- Icon-only buttons require `aria-label`
- Icon color inherits from parent via `currentColor`

### Status Emoji Mappings (used in UI text)

- Media: `🎬` (movie), `📺` (series), `🎨` (anime)
- Status: `✅` (success), `❌` (fail), `⏳` (in-progress), `·` (pending)
- Actions: `🔍` (search), `✏️` (edit)

---

## 9. Reusable UI Patterns

### Loading States

- Use `Skeleton` component with pulsing animation (1.5s ease-in-out)
- Skeleton color: `hsl(217, 33%, 20%)` base
- Preserve exact dimensions of loaded content (prevent layout shift)
- Show skeleton immediately (no artificial delay)

### Empty States

Structure: Emoji/Icon → Primary message → Secondary suggestion → Action button

```
  🔍
  找不到相關結果
  試試英文片名或簡體中文
  [調整搜尋詞]
```

### Error States

Structure: Icon → Specific message → Cause (optional) → Solution → Retry button

### Success Feedback

- `✅` icon with scale bounce animation (500ms)
- Positive tone: "修正完成!", "解析成功!"
- Auto-dismiss after 3 seconds

---

## 10. shadcn/ui Evaluation & Recommendation

### Decision: **Recommended — Adopt shadcn/ui**

The UX spec already specifies shadcn/ui as the design system foundation. It is not a traditional npm dependency but copy-paste components into `components/ui/`.

### Why shadcn/ui

1. Built on Radix UI — accessible primitives with ARIA built-in
2. Styled with Tailwind — matches our stack exactly
3. Copy-paste model — full ownership, no breaking updates
4. `clsx` + `tailwind-merge` already installed
5. Avoids "yet another Material Design" look

### Components Needed for Epic 10 (Homepage TV Wall)

| shadcn Component | Epic 10 Usage |
|---|---|
| **Carousel** | Hero Banner auto-rotating content |
| **Card** | Explore block containers |
| **Badge** | 「已有」/「已請求」availability badges |
| **Skeleton** | Loading skeleton for homepage |
| **Dialog** | Block CRUD admin modal |
| **Button** | All action buttons |

### Components Needed for Epic 11 (Advanced Search)

| shadcn Component | Epic 11 Usage |
|---|---|
| **Badge** (variant: chip) | Persistent filter chips |
| **Popover** | Filter panel dropdown |
| **Command** | Search suggestion dropdown |
| **Select** | Sort key selector |
| **Dialog** | Filter preset save/load modal |
| **Sheet** | Mobile filter bottom sheet |

### Additional Dependencies

```
# Required by shadcn/ui
pnpm add class-variance-authority
# Already installed: clsx, tailwind-merge, lucide-react
```

### Setup Notes

- Run `npx shadcn@latest init` (select: New York style, Tailwind CSS, `apps/web/src/components/ui`)
- Override default theme colors with Midnight Blue tokens from this document
- Each component is installed individually: `npx shadcn@latest add button card badge dialog carousel skeleton`

---

## 11. Tailwind Config Target State

After `0-UX-2` is complete, `apps/web/tailwind.config.js` should contain:

```js
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        primary: 'hsl(var(--bg-primary))',
        secondary: 'hsl(var(--bg-secondary))',
        tertiary: 'hsl(var(--bg-tertiary))',
        accent: {
          DEFAULT: 'hsl(var(--accent-primary))',
          hover: 'hsl(var(--accent-hover))',
          pressed: 'hsl(var(--accent-pressed))',
        },
        success: 'hsl(var(--success))',
        error: 'hsl(var(--error))',
        warning: 'hsl(var(--warning))',
        info: 'hsl(var(--info))',
        foreground: {
          DEFAULT: 'hsl(var(--text-primary))',
          secondary: 'hsl(var(--text-secondary))',
          muted: 'hsl(var(--text-muted))',
          inverse: 'hsl(var(--text-inverse))',
        },
      },
      fontFamily: {
        sans: ['Noto Sans TC', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['JetBrains Mono', 'Consolas', 'Monaco', 'monospace'],
      },
      fontSize: {
        xs: ['0.75rem', { lineHeight: '1.5' }],
        sm: ['0.875rem', { lineHeight: '1.5' }],
        base: ['1rem', { lineHeight: '1.6' }],
        lg: ['1.125rem', { lineHeight: '1.6' }],
        xl: ['1.25rem', { lineHeight: '1.6' }],
        '2xl': ['1.5rem', { lineHeight: '1.5' }],
        '3xl': ['1.875rem', { lineHeight: '1.4' }],
      },
      aspectRatio: {
        poster: '2 / 3',
        backdrop: '16 / 9',
      },
      gridTemplateColumns: {
        'media-grid': 'repeat(auto-fill, minmax(200px, 1fr))',
      },
      borderRadius: {
        sm: '4px',
        md: '8px',
        lg: '12px',
        xl: '16px',
        '2xl': '24px',
      },
      boxShadow: {
        sm: '0 1px 2px rgba(0, 0, 0, 0.3)',
        md: '0 4px 8px rgba(0, 0, 0, 0.4)',
        lg: '0 8px 16px rgba(0, 0, 0, 0.5)',
        xl: '0 12px 24px rgba(0, 0, 0, 0.6)',
        '2xl': '0 24px 48px rgba(0, 0, 0, 0.7)',
      },
      keyframes: {
        shrink: {
          '0%': { width: '100%' },
          '100%': { width: '0%' },
        },
      },
      animation: {
        shrink: 'shrink 10s linear forwards',
      },
    },
  },
  plugins: [],
};
```

---

## 12. CSS Variables Target State

After `0-UX-3` is complete, `apps/web/src/styles.css` should define:

```css
@layer base {
  :root {
    /* Background */
    --bg-primary: 222 47% 11%;
    --bg-secondary: 217 33% 17%;
    --bg-tertiary: 215 28% 23%;

    /* Accent */
    --accent-primary: 217 91% 60%;
    --accent-hover: 217 91% 70%;
    --accent-pressed: 217 91% 50%;

    /* Semantic */
    --success: 142 76% 36%;
    --error: 0 84% 60%;
    --warning: 38 92% 50%;
    --info: 200 98% 48%;

    /* Text */
    --text-primary: 0 0% 95%;
    --text-secondary: 0 0% 70%;
    --text-muted: 0 0% 50%;
    --text-inverse: 222 47% 11%;

    /* Border Radius */
    --radius: 12px;

    /* Shadows (used directly, not as HSL) */
    --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.3);
    --shadow-md: 0 4px 8px rgba(0, 0, 0, 0.4);
    --shadow-lg: 0 8px 16px rgba(0, 0, 0, 0.5);
    --shadow-xl: 0 12px 24px rgba(0, 0, 0, 0.6);
    --shadow-2xl: 0 24px 48px rgba(0, 0, 0, 0.7);
  }

  /* Reserved for future light theme */
  /* [data-theme="light"] { ... } */

  html, body, #root {
    background-color: hsl(var(--bg-primary));
    color: hsl(var(--text-primary));
  }

  a {
    color: inherit;
    text-decoration: inherit;
  }
}
```

---

## 13. Color Migration Cheat Sheet

Quick reference for `0-UX-5` (replacing hard-coded Tailwind classes):

| Current (hard-coded) | Replace With (semantic) |
|---|---|
| `bg-slate-900`, `bg-[#0f172a]` | `bg-primary` |
| `bg-slate-800` | `bg-secondary` |
| `bg-slate-700` | `bg-tertiary` |
| `text-slate-100`, `text-[#f1f5f9]` | `text-foreground` |
| `text-slate-400`, `text-slate-300` | `text-foreground-secondary` |
| `text-slate-500`, `text-gray-500` | `text-foreground-muted` |
| `border-slate-700`, `border-slate-600` | `border-tertiary` |
| `bg-blue-600`, `bg-blue-500` | `bg-accent` |
| `hover:bg-blue-700` | `hover:bg-accent-hover` |
| `bg-green-*` | `bg-success` |
| `bg-red-*`, `text-red-*` | `bg-error` / `text-error` |
| `bg-yellow-*`, `bg-orange-*` | `bg-warning` / `text-warning` |

**Rule:** Visual result must remain identical after migration. Run visual regression checks.

---

## Quick Decision Guide

| Question | Answer |
|---|---|
| Which color for a button? | `bg-accent` (primary), `bg-secondary` (secondary), `bg-error` (destructive) |
| Which font for body text? | `font-sans` (Noto Sans TC) |
| Which font for file paths? | `font-mono` (JetBrains Mono) |
| How much padding on a card? | `p-6` (standard), `p-4` (compact), `p-8` (large) |
| Which border-radius for cards? | `rounded-lg` (12px) |
| Which shadow for cards? | `shadow-md` default, `shadow-xl` on hover |
| How to define component variants? | CVA pattern (Section 7) |
| Need a Dialog/Modal? | Use shadcn `<Dialog>` after 0-UX-4 |
| Need a loading state? | Use shadcn `<Skeleton>` after 0-UX-4 |
