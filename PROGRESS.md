# PokeChamps Logger -- Progress

## Status

**Phase:** Phase 1 complete.
**Next:** Phase 2 -- Auth (email/password + Google + Discord OAuth, JWT + refresh token rotation).

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

### Phase 2 -- Auth (in progress)

#### Layer 1 -- Foundation (done)
- [x] Deps: `golang-jwt/jwt/v5`, `markbates/goth`, `go-playground/validator/v10`, `stretchr/testify`
- [x] Models: `User`, `OAuthAccount`, `RefreshToken` added to AutoMigrate
- [x] Config: `AUTH_SECRET` is now required at startup; OAuth client fields wired
- [x] `internal/testutil/db.go`: rolled-back transaction helper for integration tests

#### Layer 2 -- Email/password + JWT (done, tests pending)
- [x] `POST /auth/register` -- bcrypt hash, create user, issue tokens
- [x] `POST /auth/login` -- compare hash, issue tokens; same error for wrong email/password
- [x] `POST /auth/refresh` -- bcrypt-compare cookie against DB, rotate on match
- [x] `POST /auth/logout` -- delete refresh token row, clear cookie
- [x] `GET /api/v1/auth/me` -- returns user from JWT context
- [x] `internal/auth/token.go` -- IssueTokens, ParseAccessToken, cookie helpers
- [x] `internal/auth/middleware.go` -- JWT middleware, injects userID into context
- [ ] Integration tests: register → login → refresh → logout cycle
- [ ] Integration tests: duplicate email rejection, wrong password, missing token

#### Layer 3 -- OAuth (next)
- [ ] Goth provider setup (Google + Discord) in `cmd/api/main.go`
- [ ] `GET /auth/google` + `GET /auth/google/callback`
- [ ] `GET /auth/discord` + `GET /auth/discord/callback`
- [ ] Upsert `oauth_accounts` row on callback; link to existing user if email matches

#### Layer 4 -- Verification (done)
- [x] Cross-user isolation verified -- /auth/me returns own data only, refresh
  token for user A cannot produce a token for user B, wrong-secret middleware
  blocks valid tokens
- [x] Security review: O(n*bcrypt) refresh scan fixed -- cookie now carries
  {rowID}:{rawHex} for O(1) indexed lookup before single bcrypt compare
- [x] bcryptCost constant unified (was hardcoded 12 in Register handler)

### Phase 3 -- Team Builder
- [ ] Team CRUD endpoints (`GET /teams`, `POST /teams`, `PATCH /teams/:id`, `DELETE /teams/:id`)
- [ ] `POST /teams/:id/activate` with transaction (deactivate all others)
- [ ] DB tables: `teams`, `team_slots`, `team_slot_moves`
- [ ] Auto-save: localStorage write (300ms debounce) + server sync (1500ms debounce)
- [ ] SvelteKit team builder UI: 6 slots, species/ability/item/move autocomplete
- [ ] Active team selection UI

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
| Hosting provider (Render, Railway, Fly.io, VPS) | Not blocking Phase 1-2 |
| `training_points` exact schema | Not blocking -- ship as JSONB, tighten later |
| Refresh token: fixed 30-day TTL vs sliding window | Not blocking -- fixed TTL first |
| `our_picks` lead/back role: store in pick phase or derive from turn 1 | Blocks Phase 4 pick-phase endpoint |

---

## Docs

| File | Purpose |
|---|---|
| `docs/PRD.md` | Product requirements |
| `docs/wireframes.md` | ASCII UI mockups |
| `docs/tech-spec.md` | Data model, API contracts, auth flow, stack |
| `docs/design-system.md` | Color tokens, typography, animation rules, component conventions |
| `docs/adr/` | One ADR per major architectural decision (001-008) |
