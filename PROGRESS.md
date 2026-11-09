# PokeChamps Logger -- Progress

## Status

**Phase:** Phase 3 complete (team builder -- backend + tests + UI). OAuth login verified working end-to-end. All 4 team-builder bugs from manual testing fixed. Training points (EV-like stat allocation) + master-detail layout + fuzzy search + bits-ui adoption complete and verified in-browser.
**Next:** Phase 4 -- Match Logger.

---

## Done -- Auth Race Condition Fixes

Auth (register, login, Google OAuth) fully tested and working. Root cause was a
race: hard `window.location.href` reloads after login/OAuth re-triggered
`+layout.svelte`'s silent-refresh `onMount` concurrently with the page's own
auth check, which could invalidate a freshly-rotated refresh cookie before the
new page read it. Fixed by switching to SvelteKit `goto()` (no reload) and
gating pages on a new `authReady` promise (`frontend/src/lib/auth.svelte.ts`)
that resolves once the layout's restore attempt finishes.
- [x] `frontend/src/lib/auth.svelte.ts` -- added `authReady` promise + `markAuthReady()`
- [x] `frontend/src/routes/+layout.svelte` -- calls `markAuthReady()`; guards against clobbering a session set concurrently by the OAuth callback
- [x] `frontend/src/routes/auth/+page.svelte` -- `goto('/teams')` instead of hard reload
- [x] `frontend/src/routes/auth/callback/+page.svelte` -- `goto('/teams')` instead of hard reload; calls `markAuthReady()` on success
- [x] `frontend/src/routes/teams/+page.svelte` -- `onMount` awaits `authReady` before checking `authState.user`

## Done -- Team Builder Bug Fixes (found during manual OAuth/team-builder testing)

- [x] **Mega Pokemon no longer pickable as a team member.** `internal/handler/pokemon.go`
  `ListPokemon` filters `name NOT LIKE '%-mega%'`. Mega is a match-time state change (1
  activation/round, driven by a held mega stone), not a team-build-time species choice.
- [x] **Held-item picker.** Scraped `https://pokebase.app/pokemon-champions/items` (148 items:
  75 mega stones, 45 held items, 28 berries) via new `cmd/scrape/items.go`; seeded via
  `cmd/seed/main.go`'s new items-upsert + mega-forms steps (16 mega forms now linked). Added
  `GET /api/v1/items` (`internal/handler/pokemon.go` `ListItems`) and a held-item `<select>`
  per slot in `teams/+page.svelte`, wired to `item_id` in the PATCH payload.
- [x] **Sprites shown in team builder.** `<img>` added to the Pokemon search dropdown (small)
  and next to the selected Pokemon's type badges (large) in `teams/+page.svelte`.
- [x] **Type badge text readability fixed.** Added `text-shadow` to `.type-pip`/`.type-badge`
  in `teams/+page.svelte` (font fix, not a background-color change, per request).

---

## Done -- Training Points + Layout Redesign

Plan: `docs/adr/009-training-points-schema.md`. Pokemon Champions training points are a
66-point pool per Pokemon, spread freely across the 6 stats, capped at 32 per individual
stat, added 1:1 to the base stat (no divisor) -- confirmed with user, matches the real game
mechanic. Layout rebuilt as master-detail (slot list + summary left, full editor right,
matching the pattern `docs/design-system.md` already earmarked for this page), with
fuzzy-searchable item/move pickers, type-colored move indicators, colored per-stat summaries
(HP green, Attack red, Defense yellow, Sp.Atk blue, Sp.Def purple, Speed pink), and a
Pikalytics/PokemonDB-style stat bar graph (full + miniature versions).

- [x] `docs/tech-spec.md` -- finalized `training_points` shape + example, removed open question
- [x] `docs/adr/009-training-points-schema.md` -- ADR recording the formula/caps/bits-ui decision
- [x] Backend: `TrainingPointsJSON` type (`internal/model/seed.go`), wired into `slotInput`/`slotOut`
  and `validateSlots` (32/stat, 66 total) in `internal/handler/team.go`; `team_test.go` cases for
  the 32-per-stat and 66-total boundaries
- [x] `GET /api/v1/pokemon` and `/api/v1/items` -- `limit` query param now actually wired (was
  hardcoded to 20 before), so the frontend can preload the full non-mega roster/item catalog for
  client-side fuzzy search
- [x] Frontend foundations: `bits-ui` dependency, `frontend/src/lib/fuzzy.ts` (hand-rolled
  scorer, no extra dependency), `frontend/src/lib/typeOrder.ts`, `frontend/src/lib/stats.ts`
  (`effectiveStats`, `trainingPointsUsed`, `statColors`), `types.ts` additions
- [x] `frontend/src/lib/components/` (new dir): `Combobox.svelte`, `Select.svelte` (bits-ui
  wrappers), `MoveTypeIcon.svelte`
- [x] `frontend/src/lib/components/StatSlider.svelte` -- **not** built on bits-ui's `Slider`.
  bits-ui couples the drag-to-value math and the visual thumb/fill position to the same `max`
  prop; this UI needed them decoupled (track always renders full-length representing the fixed
  32-point scale, but dragging must hard-stop at the much smaller remaining budget). Every
  combination of bits-ui's `max` prop broke one side or the other (see chat history for the full
  trace). Solved with a native `<input type="range" min=0 max=32>`: the budget cap is enforced by
  writing the DOM's own `.value` back synchronously inside the same `input` event (no framework
  round-trip), so drag distance and visual position stay 1:1 with the true 32-point scale and the
  thumb physically can't be dragged past the budget.
- [x] `teams/+page.svelte` rewritten to master-detail (280px/1fr, 900px responsive collapse),
  all pickers migrated onto the new components
- [x] Training points UI (6 sliders w/ number input, +/-/reset/max) + live-reactive effective-stats
  summary in the left-panel mini graph, right-panel full stat-bar graph, and slot-list ability/
  item/move preview -- all update instantly on slider change, no save required to see them
- [x] Verified end-to-end in browser by user across several rounds of bug fixes (dropdown
  positioning/open-on-click, budget double-counting, slider drag/visual decoupling)

---

## Phase Checklist

### Phase 0 -- Scaffold
- [x] `.gitignore`, `.env.example`, `docker-compose.yml`
- [x] Go module init: `github.com/dreynaldis/pokechamps-logger`
- [x] Go deps: `chi/v5`, `gorm`, `gorm/driver/postgres`, `godotenv`, `goquery`
- [x] `go build ./...` passes clean
- [x] `cmd/api/main.go`, `internal/config`, `internal/database`, `internal/handler/health.go`
- [x] SvelteKit scaffold: `frontend/` with `adapter-static` SPA mode, `vite-plugin-pwa`, TypeScript, Svelte 5 runes
- [x] Docker Compose running on port 5433
- [x] `go run ./cmd/api` connects to DB, `GET /health` returns 200

### Phase 1 -- Seed Pipeline + Data Verification
- [x] `cmd/scrape`: scrapes pokebase.app → `seed/pokebase-raw.json` (310 Pokemon, rate-limited 1 req/sec)
- [x] `cmd/enrich`: hits PokeAPI for move and ability metadata → `seed/moves-enriched.json`, `seed/abilities-enriched.json`
- [x] `cmd/seed`: upserts all seed data into Postgres in a single transaction (abilities → moves → pokemon → learnsets)
- [x] GORM models: `Pokemon`, `Move`, `Ability`, `PokemonAbility`, `PokemonLearnset`, `MegaForm`, `Item`
- [x] `GET /api/v1/pokemon` -- searchable list endpoint (ILIKE, limit param)
- [x] `GET /api/v1/pokemon/:name` -- detail endpoint with preloaded abilities and learnset
- [x] CORS middleware for local SvelteKit dev server
- [x] Phase 1 verification UI: searchable list + detail panel (stats bars, abilities, moves table)
- [x] 310 Pokemon seeded and queryable. Spot-checked: types, base stats, abilities, moves all correct.
- [x] `dex_number` non-unique (regional forms and alternate forms share dex numbers)
- [x] `pokemon_abilities` join: no slot or is_hidden (not meaningful for Champions)
- [x] README with setup instructions and pokebase.app attribution
- [x] Pushed to GitHub: `github.com/Dreynaldis/pokechamps-logger`

### Phase 2 -- Auth

#### Layer 1 -- Foundation
- [x] Deps: `golang-jwt/jwt/v5`, `markbates/goth`, `go-playground/validator/v10`, `stretchr/testify`
- [x] Models: `User`, `OAuthAccount`, `RefreshToken` added to AutoMigrate
- [x] Config: `AUTH_SECRET` is now required at startup; OAuth client fields wired
- [x] `internal/testutil/db.go`: rolled-back transaction helper for integration tests

#### Layer 2 -- Email/password + JWT
- [x] `POST /auth/register` -- bcrypt-12 hash, create user, issue tokens
- [x] `POST /auth/login` -- compare hash, issue tokens; same error for wrong email/password (no enumeration)
- [x] `POST /auth/refresh` -- O(1) lookup by rowID from cookie, bcrypt-compare, rotate
- [x] `POST /auth/logout` -- delete refresh token row, clear cookie
- [x] `GET /api/v1/auth/me` -- returns user from JWT context
- [x] `internal/auth/token.go` -- IssueTokens, ParseAccessToken, cookie format: `{rowID}:{rawHex}`
- [x] `internal/auth/middleware.go` -- JWT middleware, injects userID into context
- [x] Integration tests: register → login → refresh → logout cycle (18 tests in auth_test.go)
- [x] Duplicate email rejection, wrong password, missing token, expired cookie all tested

#### Layer 3 -- OAuth
- [x] Goth provider setup (Google + Discord) in `cmd/api/main.go`; providers skipped at startup if env vars absent
- [x] `GET /auth/{provider}` + `GET /auth/{provider}/callback` via gothic
- [x] gorilla/sessions store for OAuth CSRF state cookie (SameSite=Lax)
- [x] `UpsertOAuthUser`: finds existing OAuthAccount, or links to existing User by email, or creates new User + OAuthAccount
- [x] Callback redirects to `frontend/auth/callback#access_token=<jwt>` (token in hash, never server logs)
- [x] Unit tests: new user, returning user, link-to-existing-email, two-provider same email (4 tests in oauth_test.go)

#### Layer 4 -- Cross-user isolation + security
- [x] `/auth/me` returns only requesting user's data
- [x] Wrong JWT secret blocks valid-looking token
- [x] Refresh cookie for user A cannot produce a token sub=userB
- [x] O(n*bcrypt) scan fixed: cookie carries `{rowID}:{rawHex}`, single indexed DB lookup
- [x] bcryptCost constant unified (bcrypt cost 12 ~300ms)
- [x] No user enumeration on login (wrong email and wrong password return identical error)
- [x] Cross-user isolation: all team queries filter by `user_id` from JWT context, return 404 not 403

### Phase 3 -- Team Builder

#### Layer 1 -- Models
- [x] `Team`, `TeamSlot`, `TeamSlotMove` GORM models
- [x] TeamSlot: `uniqueIndex(team_id, slot)`, `check(slot >= 1 AND slot <= 6)`
- [x] TeamSlotMove: composite PK `(team_slot_id, slot)`, `check(slot >= 1 AND slot <= 4)`
- [x] `TrainingPoints` stored as JSONB (schema finalized, see ADR 009)
- [x] Added to AutoMigrate

#### Layer 2 -- CRUD endpoints
- [x] `GET /api/v1/teams` -- list user's teams (no slots preloaded)
- [x] `POST /api/v1/teams` -- create team (name required, max 50 chars)
- [x] `GET /api/v1/teams/:id` -- full team with slots + Pokemon/Ability/Item/Move preloaded
- [x] `PATCH /api/v1/teams/:id` -- full replace: delete moves → slots, insert new (single transaction)
- [x] `DELETE /api/v1/teams/:id` -- cascading delete (explicit child order, no DDL cascade reliance)
- [x] `POST /api/v1/teams/:id/activate` -- transaction: clears all user is_active, sets target
- [x] `GET /api/v1/pokemon/:name` response now includes `id` on ability and move refs (needed by team builder)

#### Layer 3 -- Integration tests
- [x] 20 tests in team_test.go covering:
  - CRUD happy path (create, read, patch with slots/moves, delete)
  - Activation: set active, already-active stays active
  - Validation: name too long, invalid slot range, duplicate slots, missing pokemon_id, too many moves
  - Isolation: user A cannot read/patch/delete/activate user B's team (all return 404)

#### Layer 4 -- SvelteKit UI
- [x] `frontend/src/lib/api.ts` -- `api()` fetch wrapper: adds Bearer header, intercepts 401, silent refresh, retries once
- [x] `frontend/src/lib/auth.svelte.ts` -- Svelte 5 `$state` reactive auth store (user + loading); imported directly by components
- [x] `frontend/src/lib/types.ts` -- shared TypeScript types for all domain objects + draft types
- [x] `frontend/src/routes/+layout.svelte` -- sticky navbar; `onMount` silently restores session via refresh cookie on every page load
- [x] `frontend/src/routes/auth/+page.svelte` -- login/register tabs; OAuth buttons for Google/Discord
- [x] `frontend/src/routes/auth/callback/+page.svelte` -- reads `#access_token` from hash, stores in memory, clears from URL
- [x] `frontend/src/routes/teams/+page.svelte` -- team list sidebar + master-detail slot builder
  (slot list/summary left, full editor right); fuzzy-searchable Pokemon/item/move pickers
  (bits-ui `Combobox`); ability/nature `Select`; training-point sliders with live stat-bar
  graphs; explicit `markDirty()` auto-save (localStorage 300ms, server PATCH 1500ms)
- [x] Build clean (zero TypeScript errors)
- [x] Original Phase 1 data verification UI (`/`) untouched

### Phase 4 -- Match Logger
- [ ] Match CRUD: `POST /matches`, `POST /matches/:id/pick-phase`, `POST /matches/:id/turns`, `POST /matches/:id/end`
- [ ] DB tables: `matches`, `match_pick_phase`, `match_enemy_pokemon`, `turns`, `turn_actions`, `hp_events`, `secondary_effects`
- [ ] Auto-save: turn submissions immediate (no debounce)
- [ ] Mid-match detection on app boot (localStorage `match-active` + server verify)
- [ ] SvelteKit pick phase UI + turn logger UI
- [ ] Resume banner for in-progress matches

### Phase 5 -- Match History
- [ ] `GET /matches` paginated list
- [ ] `GET /matches/:id` full detail
- [ ] Match list UI: team summary, result badge
- [ ] Match detail UI: pick phase + turn-by-turn replay

### Phase 6 -- PWA + Mobile Hardening
- [ ] PWA manifest + service worker (vite-plugin-pwa)
- [ ] Tested on real mid-range Android device
- [ ] Full turn logged in under 20 seconds
- [ ] Installable as PWA

---

## Key Decisions

| Decision | Choice | ADR |
|---|---|---|
| Database | PostgreSQL | 001 |
| ORM | GORM | 002 |
| Backend | Go + Chi | 003 |
| Frontend | SvelteKit (SPA mode) | 004 |
| Auth | JWT + refresh token rotation | 005 |
| Seed pipeline | scrape + enrich + override layer | 006 |
| Seed data visibility | gitignored JSON files | 007 |
| Client persistence | localStorage buffer + server sync | 008 |
| Training points schema + bits-ui adoption | 66-pt pool, 32/stat cap, 1:1 addition; bits-ui for comboboxes | 009 |

---

## Open Questions

| Question | Blocking |
|---|---|
| Hosting provider (Render, Railway, Fly.io, VPS) | Not blocking |
| Refresh token: fixed 30-day TTL vs sliding window | Not blocking -- fixed TTL first |
| `our_picks` lead/back role: store in pick phase or derive from turn 1 | Blocks Phase 4 pick-phase endpoint |
| Google Console + Discord Developer app setup | Needed before OAuth buttons are functional |

---

## Docs

| File | Purpose |
|---|---|
| `docs/PRD.md` | Product requirements |
| `docs/wireframes.md` | ASCII UI mockups |
| `docs/tech-spec.md` | Data model, API contracts, auth flow, stack |
| `docs/design-system.md` | Color tokens, typography, animation rules, component conventions |
| `docs/adr/` | One ADR per major architectural decision (001-009) |
