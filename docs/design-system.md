# PokeChamps Logger -- Design System

## Philosophy

Futuristic minimalistic.
The UI should feel clean and purposeful -- no decorative clutter, no gratuitous animation.
Every visual element either carries information or reinforces identity.
The game itself is fast-paced and data-dense; the UI respects that by staying out of the way.

## Color Palette

### Dark Mode (default)

| Token | Value | Usage |
|---|---|---|
| `bg-base` | `#0c0c16` | Page background |
| `bg-surface` | `#13132a` | Cards, panels |
| `bg-overlay` | `#1c1c38` | Hover states, table rows |
| `bg-gradient-center` | `#1a1040` | Radial gradient focal point (see Background) |
| `border` | `#2a2a50` | Dividers, card borders |
| `text-primary` | `#e8e8f0` | Body text |
| `text-secondary` | `#9090b0` | Labels, muted text |
| `text-disabled` | `#505070` | Placeholder, empty states |
| `accent` | `#6366f1` | Primary interactive, stat bars, active states |
| `accent-hover` | `#4f52d4` | Accent hover |
| `accent-muted` | `#6366f120` | Accent tint backgrounds |
| `danger` | `#f87171` | Errors, delete actions |
| `success` | `#6bcc8a` | Success states, immune badges |

### Light Mode

| Token | Value | Usage |
|---|---|---|
| `bg-base` | `#f8f8fc` | Page background |
| `bg-surface` | `#ffffff` | Cards, panels |
| `bg-overlay` | `#f0f0fa` | Hover states, table rows |
| `bg-gradient-center` | `#e8e8ff` | Radial gradient focal point (see Background) |
| `border` | `#dcdcf0` | Dividers, card borders |
| `text-primary` | `#18182e` | Body text |
| `text-secondary` | `#5a5a7a` | Labels, muted text |
| `text-disabled` | `#b0b0cc` | Placeholder, empty states |
| `accent` | `#6366f1` | Same accent in both modes |
| `accent-hover` | `#4f52d4` | Accent hover |
| `accent-muted` | `#6366f115` | Accent tint backgrounds |
| `danger` | `#dc2626` | Errors |
| `success` | `#16a34a` | Success states |

### Pokemon Type Colors

Type badges use fixed colors regardless of light/dark mode.
Text on all type badges is always white (`#ffffff`).

| Type | Color |
|---|---|
| Normal | `#A8A878` |
| Fire | `#F08030` |
| Water | `#6890F0` |
| Electric | `#F8D030` |
| Grass | `#78C850` |
| Ice | `#98D8D8` |
| Fighting | `#C03028` |
| Poison | `#A040A0` |
| Ground | `#E0C068` |
| Flying | `#A890F0` |
| Psychic | `#F85888` |
| Bug | `#A8B820` |
| Rock | `#B8A038` |
| Ghost | `#705898` |
| Dragon | `#7038F8` |
| Dark | `#705848` |
| Steel | `#B8B8D0` |
| Fairy | `#EE99AC` |

## Background

The background is a radial gradient centered on the viewport, giving depth without motion.
A large low-opacity Pokeball SVG watermark sits behind all content, fixed to the viewport center.

```css
/* Dark mode */
background: radial-gradient(ellipse at center, #1a1040 0%, #0c0c16 65%);

/* Light mode */
background: radial-gradient(ellipse at center, #e8e8ff 0%, #f8f8fc 65%);
```

**Pokeball watermark:**
- Size: `min(50vw, 500px)` -- caps at 500px on large screens
- Position: fixed center, `z-index: 0`, behind all layout
- Opacity: `0.04` dark mode / `0.06` light mode
- No animation, no rotation

## Typography

System font stack -- no custom font loaded.
Loading a font is a network request; system fonts render instantly and look native.

```css
font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
```

| Scale | Size | Weight | Usage |
|---|---|---|---|
| `text-xs` | `0.72rem` | 400 | Type badges, small tags |
| `text-sm` | `0.82rem` | 400 | Table rows, ability descriptions |
| `text-base` | `0.9rem` | 400 | Body, list items |
| `text-md` | `1rem` | 600 | Card headings, nav links |
| `text-lg` | `1.3rem` | 700 | Pokemon name in detail panel |
| `text-xl` | `1.5rem` | 700 | Page title, logo wordmark |

## Spacing

Base unit: `4px`.
All spacing is multiples of 4: `4 8 12 16 24 32 48`.
Padding inside cards: `16px`.
Gap between cards: `12px`.
Navbar height: `56px`.

## Border Radius

| Context | Radius |
|---|---|
| Cards, panels | `10px` |
| Buttons, inputs | `6px` |
| Type badges | `12px` (pill) |
| Tags (Pivot, +Eff) | `4px` |
| Stat bars | `4px` |

## Shadows

Dark mode uses no box-shadow -- the dark background provides sufficient contrast via border.
Light mode cards get a single light shadow:

```css
/* Light mode cards only */
box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
```

No glow effects, no multiple-layer shadows.

## Animation

**Rule: smooth and purposeful. Animation should make the UI feel responsive and alive, not theatrical.**

Preferred properties (GPU-composited, no layout thrashing): `opacity`, `transform`, `background-color`, `border-color`, `color`.
Avoid animating `width`, `height`, `top`, `left`, `padding`, `margin` -- these cause layout recalculation.
Exception: stat bars use `width` transition because the fill is data-meaningful and happens once on load.

Standard durations:
- `100ms ease-out` -- hover state changes (buttons, rows, nav links)
- `150ms ease-out` -- focus rings, border color changes, small interactive feedback
- `200ms ease-out` -- content appearing (detail panel fade-in, dropdown open)
- `300ms ease-out` -- stat bar fill on Pokemon select (longer feels more satisfying here)
- `200ms ease-in-out` -- theme toggle icon rotation

Allowed:
- Detail panel content fade-in when selecting a Pokemon: `opacity 0 → 1` + subtle `translateY(4px) → 0`
- Stat bar width fill on Pokemon select
- Button/row hover: `background-color` shift
- Theme toggle: icon crossfade or rotation
- Dropdown/menu open: `opacity` + subtle `translateY`
- Active nav link underline slide
- Toast notifications sliding in from top/bottom

Not allowed:
- Scroll-triggered animations (elements flying in as you scroll down a page)
- Entrance animations on every page load
- Looping or auto-playing animations on idle UI
- Bounce, elastic, or spring easing on data UI
- Parallax effects

## Third-Party UI Components

Third-party component libraries are allowed and preferred over browser built-ins.
Browser native UI (alert, confirm, prompt dialogs, `<select>` dropdowns, `<details>`) feels archaic and is not styleable -- replace them.

**Preferred library: [bits-ui](https://bits-ui.com)** -- headless, unstyled primitives for Svelte.
It gives accessible behaviour (keyboard nav, ARIA, focus trap) without imposing a visual style.
We own the CSS; bits-ui owns the interaction logic.

Use it for:
- Modals / confirmation dialogs (replaces `window.confirm`)
- Dropdown menus and select inputs (replaces `<select>`)
- Tooltips
- Popovers (e.g. move detail on hover)
- Combobox / autocomplete search

**Avoid:**
- Full-kit libraries (shadcn-svelte, Skeleton UI, etc.) that bundle a design system on top of headless primitives -- we already have our own design tokens.
- Any component library that requires wrapping everything in a provider or global store.

For notifications/toasts, use [svelte-sonner](https://github.com/wobsoriano/svelte-sonner) -- minimal, no style lock-in.

## Layout

### Navbar

Height: `56px`. Sticky top. Background inherits the gradient (same start color as `bg-base`), with a 1px bottom border using `border` token.

```
[ Pokeball icon + "PokeChamps" ] ---- [ Team | Battles | History ] ---- [ ☀/🌙 ] [ Log in ] [ Sign up ]
```

- Logo: Pokeball SVG (24px) + wordmark. Clicking navigates to `/`.
- Nav links: hidden in Phase 1-2 (no routes yet). Added now as layout scaffold, rendered as disabled or omitted until Phase 3.
- Theme toggle: icon button, no label. Sun icon in dark mode, moon icon in light mode.
- Auth buttons: "Log in" (ghost style) + "Sign up" (filled accent). Collapse to avatar + dropdown once logged in.

### Page Layout

Max width: `1280px`, centered, `1rem` side padding.
Below the navbar, pages are full-height with `height: calc(100vh - 56px)`.

### Data Verification Page (Phase 1 -- current)

Two-column grid: `280px` left panel + `1fr` right panel.
- Left panel: search input fixed at top, Pokemon list scrolls below it (`overflow-y: auto`).
- Right panel: Pokemon name + types fixed at top, base stats + abilities fixed below, **moves table scrolls** (`overflow-y: auto`, `max-height` fills remaining space).

This layout will be replaced in Phase 3 by the actual team builder, but the scroll model carries forward.

## Component Conventions

**Buttons**

| Variant | Style |
|---|---|
| Primary (filled) | `accent` background, white text |
| Ghost | Transparent background, `accent` text, `border` border |
| Danger | `danger` background, white text |
| Icon | 36px square, transparent, `text-secondary` icon, hover `bg-overlay` |

**Inputs**

Single style: `bg-surface` background, `border` border, `text-primary` text, `accent` focus ring (2px).
No floating labels -- use `<label>` above the input.

**Cards**

`bg-surface` background, `border` border, `10px` radius, `16px` padding.
No hover lift effect on cards -- cards are containers, not interactive targets.

**Tags**

Small inline labels for move metadata and status.
`bg-overlay` background, `text-secondary` text, `4px` radius, `2px 6px` padding, `0.68rem` font size.
Semantic overrides: Immune (`success` tint), Pivot (`accent` tint).

## Icon Strategy

No icon library dependency.
Use inline SVG for the 5-10 icons needed (Pokeball logo, sun, moon, search, menu).
This avoids loading an entire icon set for a handful of icons.

## Dark / Light Mode Implementation

Toggle stored in `localStorage` under key `theme`.
Applied as a class on `<html>`: `class="dark"` or `class="light"`.
CSS custom properties (variables) defined under `:root.dark` and `:root.light`.
Default on first visit: `dark`.

No system preference detection (`prefers-color-scheme`) in v1 -- manual toggle only.
Can be added later without breaking the implementation.

## What This Design Is Not

- Not a Pokedex -- avoid Pokedex-style red/white color schemes.
- Not anime-themed -- no pixel art, no cartoon sprites in the UI chrome.
- Not dashboard-heavy -- no donut charts, no KPI tiles in Phase 1-3.
- Not mobile-first -- desktop is primary for Phase 1-3; mobile is Phase 6.
