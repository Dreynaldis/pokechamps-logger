# ADR 004: Frontend -- SvelteKit (SPA mode) over React + Vite

**Status:** Accepted   ·   **Date:** 2026-06-27

## Context

The frontend is a mobile-primary PWA where users log battle turns in near real-time.
The most complex screen is the turn logger: two active slots per side, each with a move-or-switch mutex, HP event entries, and conditional secondary effect prompts.
All pages are auth-gated, so there is no SEO requirement and no server-side rendering benefit.
The developer has React experience and is open to learning a new frontend framework.

## Decision

Use SvelteKit with SSR disabled (`adapter-static`, `ssr: false` in `svelte.config.js`).
Use `vite-plugin-pwa` for the PWA manifest and service worker.
Use Svelte stores (`writable`, `derived`) for auth token state and active team state.
TypeScript via `<script lang="ts">` in `.svelte` files.

## Options Considered

### Option A -- SvelteKit (SPA mode)
- Pros: Svelte compiles components to vanilla JavaScript at build time -- no virtual DOM runtime is shipped, producing a smaller bundle (faster parse and execute on mid-range Android); native reactivity (`let count = 0` is reactive by default, `$:` for derived values) reduces boilerplate for the turn logger's complex local state compared to React hooks; Svelte stores are a simpler mental model than React context for cross-component state; `vite-plugin-pwa` integrates identically to a plain Vite project; SvelteKit's file-based routing works in SPA mode without SSR; genuinely different paradigm from React, making it a high learning-value choice.
- Cons: smaller ecosystem than React -- fewer pre-built mobile UI component libraries; fewer community examples for PWA-specific patterns; the developer will be learning Svelte's reactivity model at the same time as building the app.

### Option B -- React + Vite
- Pros: developer already has React experience; largest frontend ecosystem; most community resources for PWA, autocomplete, and mobile touch patterns.
- Cons: no meaningful learning gain; React hooks (`useState`, `useEffect`, `useContext`) add boilerplate for the turn logger's local state that Svelte handles natively; larger runtime bundle shipped to the browser.

### Option C -- Vue 3 + Vite
- Pros: Composition API is clean; good documentation; single-file components similar to Svelte; smaller bundle than React.
- Cons: less learning differentiation than Svelte from the developer's perspective; intermediate ecosystem coverage compared to React (larger) and Svelte (simpler).

## Consequences

- **Positive:** smaller JavaScript bundle improves mobile performance on the primary target device; Svelte's reactivity model is a better fit for the turn logger's stateful UI; learning Svelte is high value alongside learning Go.
- **Negative:** fewer pre-built UI component libraries means more custom component work for the gamified slot layout; community resources for Svelte PWA patterns are less abundant than React; SvelteKit's SPA mode (disabling SSR on a framework designed for SSR) requires explicit config and has occasional rough edges.

## Related Docs

- Tech spec section 3.2 (frontend stack): `docs/tech-spec.md`
- Tech spec section 3.3 (architecture -- codegen bridge): `docs/tech-spec.md`
- Tech spec section 4 (alternatives): `docs/tech-spec.md`
- ADR 003 (Go backend -- OpenAPI type bridge): `docs/adr/003-backend-go-chi.md`
