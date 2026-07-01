# PokeChamps Logger -- Progress & Decisions

## Status
**Phase:** Phase 0 complete. Next: Phase 1 (seed pipeline).

### Phase 0 Checklist
- [x] `.gitignore`, `.env.example`, `docker-compose.yml`
- [x] Go module init: `github.com/dreynaldis/pokechamps-logger`
- [x] Go deps installed: `chi/v5`, `gorm`, `gorm/driver/postgres`, `godotenv`
- [x] `go build ./...` passes clean
- [x] Files: `cmd/api/main.go`, `internal/config/config.go`, `internal/database/database.go`, `internal/handler/health.go`
- [x] SvelteKit scaffold: `frontend/` with `adapter-static` SPA mode, `vite-plugin-pwa`, TypeScript
- [x] `npm install` in `frontend/` -- clean on Node 20
- [x] Docker Compose running on port 5433 (5432 reserved for local Postgres install)
- [x] `go run ./cmd/api` connects to DB, `GET /health` returns 200

## What's Done
- Full design session (grill-with-docs) to lock requirements.
- PRD written and reviewed: `docs/PRD.md`
- Wireframes written (ASCII mockups): `docs/wireframes.md`
- Tech spec written and reviewed: `docs/tech-spec.md`
- Stack revised: Go + Chi + GORM (backend), SvelteKit SPA (frontend), OpenAPI codegen for type bridge.
- Phases expanded from 3 coarse phases to 7 granular phases (Phase 0-6) with explicit exit conditions.
- ADRs written: `docs/adr/` (001 database, 002 ORM, 003 backend, 004 frontend, 005 auth, 006 seed pipeline, 007 seed data visibility).

## Key Decisions Locked

### Product
- Web-based, PWA-compatible. Mobile is the primary logging device during matches.
- Target user: online Pokemon Champions players. Not tournament players.
- Auth: email + password + social login (Google, Discord).
- Multiple teams per user; one active at a time (mirrors in-game lock before matchmaking).
- Battle format: doubles (2v2), 6 registered, bring 4, 2 lead + 2 back.

### Team Builder
- Each Pokemon slot: species, ability, held item, nature, training points, up to 4 moves.
- Minimum to save: 6 Pokemon each with at least 1 move. Held item and full 4 moves not required.
- Mega form is not a separate field -- it is derived from the held mega stone (e.g. Charizard + Charizardite Y = Mega Charizard Y). Mega eligibility implied by species.

### Battle Logger
- No separate Turn 0. Leads for both sides are entered at the start of Turn 1.
- Pick phase: user selects their 4 from active team's 6. Enemy's 6 entry is optional (time-pressured).
- Turn 1+: up to 4 actions per turn (2 per side, priority-ordered).
- Per action: move OR switch -- mutually exclusive. Pivot moves (Flip Turn, Volt Switch, U-turn, Parting Shot) are logged as a move; the forced switch-in is a secondary effect prompt.
- Enemy also has a switch action (not move-only).
- Every HP-related event is a separate discrete entry (primary damage, recoil, contact ability damage, etc.) -- each prompts "which Pokemon, how much HP% lost."
- Secondary effects prompted only when the move/ability/item has them.
- Match ends: win / loss / concede. Concede stored as loss in history list.

### UI / UX
- Color convention: blue border = your Pokemon, red border = enemy Pokemon. Applies everywhere (slots, action lines in match detail).
- Turn logger layout: `[bench small] [active big] [action buttons]` per row, mirrored for enemy side. Center divider separates the two sides.
- Match history card: your team (blue icons) | VS | enemy team (red icons) + VICTORY/DEFEAT badge.
- Action lines in match detail: `[B: Charizard] → Air Slash → [R: Landorus-T]`

### Tech Stack (decided in tech spec)
- Frontend: SvelteKit (SPA mode, SSR disabled) + vite-plugin-pwa.
- Backend: Go + Chi (HTTP router) + GORM.
- Database: PostgreSQL. JSONB used for flexible columns (training_points, secondary_effects.detail, pokemon.stats, pokemon.types).
- Auth: golang-jwt (JWT access token 15 min) + markbates/goth (Google + Discord OAuth) + refresh token (30 days, HttpOnly cookie, rotated on use).
- Autocomplete: PostgreSQL `pg_trgm` GIN index -- no separate search service needed at 310 Pokemon scale.
- Type bridge: swaggo generates OpenAPI spec from Go annotations; openapi-typescript generates TypeScript types for the SvelteKit frontend.

### Data Sources
- Pokemon learnsets, abilities, move stats (power/accuracy/pp): pokebase.app scrape (310 Pokemon).
- Move metadata (priority, target, secondary effect flags, effect_chance): PokeAPI.
- Champions override layer: `seed/champions-overrides.json` -- manually maintained each patch cycle (~monthly).
- Fallback if pokebase data unavailable: free-text move entry (no filtering).
- Pokebase.app: no contact method found (no email or Discord). Proceeding with a rate-limited scrape (1 req/sec); credit them in the app footer.
- Seed JSON files (`seed/*.json`) are gitignored -- not committed to the repo. Production DB is the durable snapshot.

### Database
- PostgreSQL. Relational schema for all structured data; JSONB only where schema is genuinely variable.
- Match data is private per user. No sharing in v1.
- No automatic expiry in v1. Retention policy revisited once storage costs are known.

### Performance
- Autocomplete search: results within 100ms per keystroke via pg_trgm GIN index.
- Turn input target: full turn logged under 20 seconds on mobile.
- Validated on mid-range Android hardware before hardening.

## Open Questions

| Question | Status |
|---|---|
| Hosting provider (Render, Railway, Fly.io, VPS) | Open -- not blocking Phase 1-2 |
| `training_points` exact schema | Open -- ship as free JSONB, tighten once Champions UI is confirmed |
| Ability slot data from pokebase.app | Open -- check during scrape; fallback to PokeAPI ability list |
| Refresh token: fixed 30-day TTL vs sliding window | Open -- ship fixed TTL first |
| `our_picks` lead/back role: store in pick phase or derive from turn 1 | Open -- needs decision before pick-phase endpoint is implemented |
| Retention policy | Open -- no auto-expiry in v1, revisit post-launch |

## What's Next (in order)

1. **Phase 0** -- repo scaffold: Go module, SvelteKit init, Docker Compose, DB connection, first migration, `/health` endpoint.
3. **Phase 1** -- seed pipeline: scrape pokebase.app, enrich from PokeAPI, apply overrides, load DB, verify data.
4. **Phase 2** -- auth: email/password + Google + Discord OAuth, JWT + refresh token rotation.
5. **Phase 3** -- team builder: CRUD endpoints + SvelteKit team builder UI.
6. **Phase 4** -- match logging: pick phase, turn logger, match-end endpoints + UI.
7. **Phase 5** -- match history: list + detail views.
8. **Phase 6** -- PWA hardening: manifest, service worker, mobile performance validation.

## Post-MVP (explicitly deferred)
- Visual health bar / damage bar in match detail (data already collected via HP% events).
- Team stats: win rate, most-used leads, most-encountered enemy (threshold TBD).
- Speed tuning and stat distribution review tools.
- Export or share match logs.

## Files
- `docs/PRD.md` -- full product requirements document.
- `docs/wireframes.md` -- ASCII mockups for all key screens.
- `docs/tech-spec.md` -- data model, API contracts, auth flow, tech stack decisions.
- `docs/adr/` -- ADRs (none yet -- to be written after tech spec review).
