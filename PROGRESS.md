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

### Phase 2 -- Auth (next)
- [ ] `POST /auth/register` -- email + password
- [ ] `POST /auth/login`
- [ ] `POST /auth/refresh` -- refresh token rotation (HttpOnly cookie)
- [ ] `POST /auth/logout`
- [ ] `GET /auth/me`
- [ ] Google OAuth via `markbates/goth`
- [ ] Discord OAuth via `markbates/goth`
- [ ] JWT middleware applied to all protected route groups
- [ ] DB tables: `users`, `oauth_accounts`, `refresh_tokens`
- [ ] Cross-user isolation verified (user A's token rejected on user B's resources)
- [ ] Security review before shipping

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
