# PokeChamps Logger -- Progress

## Status

**Phase:** Phase 3 complete (team builder -- backend + tests + UI).
**Next:** Phase 4 -- Match Logger.

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
- [x] `TrainingPoints` stored as JSONB (schema TBD)
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
- [x] `frontend/src/routes/teams/+page.svelte` -- team list sidebar + 6-slot builder; Pokemon autocomplete (200ms debounce); ability/nature/move selects per slot; explicit `markDirty()` auto-save (localStorage 300ms, server PATCH 1500ms)
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

---

## Open Questions

| Question | Blocking |
|---|---|
| Hosting provider (Render, Railway, Fly.io, VPS) | Not blocking |
| `training_points` exact schema | Not blocking -- ship as JSONB, tighten later |
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
| `docs/adr/` | One ADR per major architectural decision (001-008) |
