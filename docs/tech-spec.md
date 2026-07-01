# Tech Spec: PokeChamps Logger

**Status:** Active  ·  **Author:** Dion  ·  **Date:** 2026-06-26

---

## 1. Summary

PokeChamps Logger is a PWA-compatible web app where authenticated users build Pokemon teams and log every turn of a Pokemon Champions match.
Go + Chi backend, SvelteKit SPA frontend, PostgreSQL database.
Reference data (Pokemon, moves, abilities, items, learnsets) is seeded from pokebase.app and PokeAPI, with a Champions-specific override layer maintained in the repo.
All match data is private per user; no public sharing in v1.

---

## 2. Goals & Non-Goals

**Goals:**
- Team builder: save a team (6 Pokemon, min 1 move each) and mark one team as active.
- Match logger: log pick phase, leads, and every turn action (move or switch) including HP% events and secondary effects, in under 20 seconds per turn on mobile.
- Match history: list view and turn-by-turn detail view for past matches.
- Autocomplete Pokemon/move search within 100ms per keystroke.
- Auth: email + password, Google OAuth, Discord OAuth.

**Non-goals (v1):**
- Damage calculation (HP% is user-reported, not computed).
- Match sharing between users.
- Offline-first support (PWA shell loads fast but data requires a network connection).
- Learnset validation if pokebase.app data is unavailable (fallback: free-text move entry).
- Tournament mode.

---

## 3. Stack

| Layer | Choice |
|---|---|
| Backend | Go + Chi (HTTP router) + GORM |
| Frontend | SvelteKit SPA (SSR disabled, adapter-static) + Svelte 5 runes |
| Database | PostgreSQL -- JSONB for flexible columns, `pg_trgm` for autocomplete |
| Auth | `golang-jwt`, `markbates/goth` (OAuth), bcrypt cost 12 |
| UI components | bits-ui (headless primitives), svelte-sonner (toasts) |
| Type bridge | swaggo → openapi-typescript (Go annotations → TS types) |

Decisions and rationale for each choice live in `docs/adr/`.

---

## 4. Architecture

```
Browser (SvelteKit SPA + PWA manifest)
        |
        | HTTPS / JSON REST
        v
Go API  (Chi router -- auth, teams, matches, reference data)
        |
        | GORM
        v
PostgreSQL
        ^
        |
Seed pipeline (one-time, run at deploy):
  go run ./cmd/scrape   → seed/pokebase-raw.json
  go run ./cmd/enrich   → seed/moves-enriched.json, seed/abilities-enriched.json
  go run ./cmd/seed     → upserts all of the above into the DB
  seed/champions-overrides.json  (manually maintained per patch cycle)
```

The SvelteKit build is a static output served from a CDN or the same host as the API.
No BFF, no GraphQL -- REST is sufficient for this data shape.

---

## 5. Data Model

All PKs: `uuid DEFAULT gen_random_uuid()`.
All timestamps: `timestamptz NOT NULL DEFAULT now()`.

### 5.1 Reference / Seed Tables

Read-only at runtime. Written once by the seed pipeline, updated on each Champions patch.

```sql
pokemon (
  id           uuid PK,
  dex_number   int  NOT NULL,              -- indexed, not unique (forms share dex numbers)
  name         text UNIQUE NOT NULL,       -- slug e.g. "hisuian-samurott"
  display_name text NOT NULL,
  sprite_url   text,
  types        jsonb NOT NULL,             -- ["dragon", "ground"]
  base_stats   jsonb NOT NULL              -- {hp, attack, defense, sp_atk, sp_def, speed}
)

abilities (
  id                   uuid PK,
  name                 text UNIQUE NOT NULL,
  display_name         text NOT NULL,
  short_effect         text,
  grants_immunity_type text               -- e.g. "ground" (Levitate), used for immunity prompts
)

pokemon_abilities (
  pokemon_id uuid NOT NULL REFERENCES pokemon(id),
  ability_id uuid NOT NULL REFERENCES abilities(id),
  PRIMARY KEY (pokemon_id, ability_id)
)

moves (
  id                   uuid PK,
  name                 text UNIQUE NOT NULL,
  display_name         text NOT NULL,
  type                 text NOT NULL,
  category             text NOT NULL,     -- "physical" | "special" | "status"
  power                int,               -- null for status moves
  accuracy             int,               -- null for moves that never miss
  pp                   int NOT NULL,
  priority             int NOT NULL DEFAULT 0,
  target               text NOT NULL,     -- "selected-pokemon" | "all-opponents" | "user" | etc.
  short_effect         text,
  effect_chance        int,
  has_secondary_effect boolean NOT NULL DEFAULT false,
  is_pivot             boolean NOT NULL DEFAULT false  -- Flip Turn, Volt Switch, U-turn, Parting Shot
)

pokemon_learnsets (
  pokemon_id uuid NOT NULL REFERENCES pokemon(id),
  move_id    uuid NOT NULL REFERENCES moves(id),
  PRIMARY KEY (pokemon_id, move_id)
)

items (
  id                    uuid PK,
  name                  text UNIQUE NOT NULL,
  display_name          text NOT NULL,
  category              text NOT NULL,
  short_effect          text,
  field_effect_duration int             -- extra turns granted (Light Clay, Damp Rock, etc.)
)

mega_forms (
  id              uuid PK,
  base_pokemon_id uuid NOT NULL REFERENCES pokemon(id),
  item_id         uuid NOT NULL REFERENCES items(id),
  form_name       text NOT NULL,
  sprite_url      text,
  UNIQUE (base_pokemon_id, item_id)
)
```

### 5.2 User & Auth Tables

```sql
users (
  id            uuid PK,
  email         text UNIQUE NOT NULL,
  password_hash text,         -- null for OAuth-only accounts
  created_at    timestamptz,
  updated_at    timestamptz
)

oauth_accounts (
  id                  uuid PK,
  user_id             uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider            text NOT NULL,        -- "google" | "discord"
  provider_account_id text NOT NULL,
  created_at          timestamptz,
  UNIQUE (provider, provider_account_id)
)

refresh_tokens (
  id         uuid PK,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL,     -- bcrypt hash of the raw 32-byte random token
  expires_at timestamptz NOT NULL,
  created_at timestamptz
)
```

### 5.3 Team Tables

```sql
teams (
  id         uuid PK,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       text NOT NULL,
  is_active  boolean NOT NULL DEFAULT false,
  created_at timestamptz,
  updated_at timestamptz
)

team_slots (
  id              uuid PK,
  team_id         uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  slot            int  NOT NULL CHECK (slot BETWEEN 1 AND 6),
  pokemon_id      uuid NOT NULL REFERENCES pokemon(id),
  ability_id      uuid REFERENCES abilities(id),
  item_id         uuid REFERENCES items(id),
  nature          text,
  training_points jsonb,    -- EV-equivalent; exact shape TBD once Champions UI is confirmed
  created_at      timestamptz,
  UNIQUE (team_id, slot)
)

team_slot_moves (
  id           uuid PK,
  team_slot_id uuid NOT NULL REFERENCES team_slots(id) ON DELETE CASCADE,
  move_id      uuid NOT NULL REFERENCES moves(id),
  slot         int  NOT NULL CHECK (slot BETWEEN 1 AND 4),
  UNIQUE (team_slot_id, slot)
)
```

Active team constraint enforced at the application layer: setting a team active wraps in a transaction that clears all other `is_active` flags for that user first.

### 5.4 Match Tables

```sql
matches (
  id           uuid PK,
  user_id      uuid NOT NULL REFERENCES users(id),
  team_id      uuid NOT NULL REFERENCES teams(id),
  result       text CHECK (result IN ('win', 'loss')),  -- null while in progress
  started_at   timestamptz,
  completed_at timestamptz
)

match_pick_phase (
  id         uuid PK,
  match_id   uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE UNIQUE,
  our_picks  jsonb NOT NULL
  -- [{team_slot_id, role: "lead"|"back"}] -- 4 entries; role confirmed at turn 1
)

match_enemy_pokemon (
  id                 uuid PK,
  match_id           uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  pokemon_id         uuid REFERENCES pokemon(id),  -- null until identified
  position           int,
  identified_at_turn int
)

turns (
  id          uuid PK,
  match_id    uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  turn_number int NOT NULL,
  created_at  timestamptz,
  UNIQUE (match_id, turn_number)
)

turn_actions (
  id                         uuid PK,
  turn_id                    uuid NOT NULL REFERENCES turns(id) ON DELETE CASCADE,
  priority_order             int  NOT NULL,
  side                       text NOT NULL CHECK (side IN ('us', 'enemy')),
  our_team_slot_id           uuid REFERENCES team_slots(id),
  enemy_pokemon_id           uuid REFERENCES match_enemy_pokemon(id),
  action_type                text NOT NULL CHECK (action_type IN ('move', 'switch')),
  move_id                    uuid REFERENCES moves(id),
  outcome                    text CHECK (outcome IN ('hit', 'miss', 'protected', 'immune')),
  has_priority               boolean NOT NULL DEFAULT false,
  switch_in_our_slot_id      uuid REFERENCES team_slots(id),
  switch_in_enemy_pokemon_id uuid REFERENCES match_enemy_pokemon(id)
  -- pivot moves logged as move; the forced switch-in goes into secondary_effects
)

hp_events (
  id                      uuid PK,
  turn_action_id          uuid NOT NULL REFERENCES turn_actions(id) ON DELETE CASCADE,
  event_order             int  NOT NULL,
  target_our_slot_id      uuid REFERENCES team_slots(id),
  target_enemy_pokemon_id uuid REFERENCES match_enemy_pokemon(id),
  hp_percent_lost         int NOT NULL CHECK (hp_percent_lost BETWEEN 1 AND 100)
)

secondary_effects (
  id             uuid PK,
  turn_action_id uuid NOT NULL REFERENCES turn_actions(id) ON DELETE CASCADE,
  effect_type    text NOT NULL,
  -- "burn"|"freeze"|"paralysis"|"sleep"|"poison"|"bad_poison"|"flinch"|"confusion"
  -- "stat_change"     detail: {stat, stages, target_side, target_pokemon_id}
  -- "weather"         detail: {type: "rain"|"sun"|"sand"|"hail", duration_turns}
  -- "field_effect"    detail: {type: "trick_room"|"tailwind"|"reflect"|..., duration_turns, side}
  -- "mega_evolution"  detail: {side, actor_ref, form_name}
  -- "pivot_switch_in" detail: {switch_in_our_slot_id?, switch_in_enemy_pokemon_id?}
  detail jsonb
)
```

---

## 6. API Contracts

Base path: `/api/v1`.
All endpoints require `Authorization: Bearer <access_token>` unless marked **public**.

### Auth

| Method | Path | Notes |
|---|---|---|
| POST | `/auth/register` | **public.** `{email, password}` → `{user, accessToken}` + refresh cookie |
| POST | `/auth/login` | **public.** `{email, password}` → `{user, accessToken}` + refresh cookie |
| POST | `/auth/refresh` | Reads refresh cookie, rotates it, returns new `{accessToken}` |
| POST | `/auth/logout` | Deletes refresh token row. Returns 204 |
| GET | `/auth/google` | **public.** Starts Google OAuth redirect |
| GET | `/auth/google/callback` | **public.** Exchanges code, redirects to frontend |
| GET | `/auth/discord` | **public.** |
| GET | `/auth/discord/callback` | **public.** |
| GET | `/auth/me` | Returns current user |

### Teams

| Method | Path | Notes |
|---|---|---|
| GET | `/teams` | All teams for current user, with full slots + moves |
| POST | `/teams` | `{name}` -- creates empty team |
| PATCH | `/teams/:id` | `{name?, slots?}` -- upserts name and/or all slot data in one request |
| DELETE | `/teams/:id` | 204 |
| POST | `/teams/:id/activate` | Sets this team active; deactivates all others in a transaction |

`PATCH /teams/:id` slots body:
```json
[{
  "slot": 1, "pokemon_id": "uuid", "ability_id": "uuid",
  "item_id": "uuid", "nature": "timid", "training_points": {},
  "moves": [{ "slot": 1, "move_id": "uuid" }]
}]
```
Server validates: 6 slots, each with at least 1 move, ability belongs to that Pokemon, moves belong to that Pokemon's learnset.

### Matches

| Method | Path | Notes |
|---|---|---|
| GET | `/matches` | `?page&limit` -- paginated list with team summary, result |
| POST | `/matches` | Creates match using active team. Fails if no active team |
| GET | `/matches/:id` | Full detail: pick phase + all turns + HP events + secondary effects |
| POST | `/matches/:id/pick-phase` | `{our_picks, enemy_pokemon?}` |
| POST | `/matches/:id/turns` | `{actions: [...]}` -- logs one full turn |
| POST | `/matches/:id/end` | `{result: "win"|"loss"}` |

### Reference Data (read-only)

| Method | Path | Notes |
|---|---|---|
| GET | `/pokemon` | `?q=char&limit=10` -- trigram search on display_name |
| GET | `/pokemon/:name` | Full detail: abilities, learnset with move stats |
| GET | `/moves` | `?q=heat&pokemon_id=uuid` -- filtered to learnset if pokemon_id provided |
| GET | `/items` | `?q=char` |

---

## 7. Auth Flow

**Email/password:**
1. Register: bcrypt-hash password at cost 12, insert `users` row.
2. Login: compare hash, issue tokens on match.
3. Access token: JWT signed with `AUTH_SECRET`, 24-hour expiry, `{sub: userId}`.
4. Refresh token: 32 random bytes, bcrypt-hashed, stored in `refresh_tokens`. Sent as `HttpOnly; Secure; SameSite=Strict` cookie. 30-day expiry.
5. Refresh: read cookie, find matching hash, delete old row, issue new access token + new refresh token (rotation).
6. Logout: delete refresh token row. Client drops access token from memory.

**OAuth (Google / Discord):**
1. `/auth/google` redirects to provider with `state` param (CSRF protection).
2. Callback validates `state`, exchanges code for profile via Goth adapter.
3. Find `oauth_accounts` row by `(provider, provider_account_id)`. If found, load linked user. If not, create `users` + `oauth_accounts` rows.
4. Issue same JWT + refresh cookie as email flow.
5. Redirect to `/dashboard#token=<accessToken>`. SPA reads from hash, stores in memory (Svelte `$state`), clears hash from URL.

**Client token storage:** access token in memory only. Refresh token in HttpOnly cookie. A fetch wrapper catches 401s, calls `/auth/refresh`, retries the original request.

---

## 8. Client-side Persistence

See ADR 008 for the full decision (why localStorage over Redis, IndexedDB, sessionStorage, etc.).

**localStorage keys:**

| Key | Contents | Cleared when |
|---|---|---|
| `team-draft:new` | Unsaved new team JSON | First successful `POST /teams` |
| `team-draft:{id}` | Existing team edits | Successful `PATCH /teams/:id` |
| `match-active` | `{ matchId, status, currentTurn, localTurns[] }` | Match completed or abandoned |

**Sync timing:**

| Flow | localStorage write | Server sync |
|---|---|---|
| Team field change | 300ms debounce | 1500ms debounce |
| Pick phase selection | Immediate | Immediate |
| Turn submission | Immediate | Immediate |
| Match status change | Immediate | Immediate |

On every app boot: check `match-active`, verify against server, show resume banner if confirmed in-progress.

---

## 9. Security

- Passwords: bcrypt cost 12. Never logged or returned in responses.
- Refresh tokens: stored as bcrypt hashes. Raw token only in HttpOnly cookie -- JS cannot read it.
- `AUTH_SECRET` and all OAuth secrets: env vars, never committed.
- OAuth `state` param validated on callback (CSRF protection).
- All data endpoints scope to `WHERE user_id = <jwt sub>`. No cross-user access.
- JWT middleware applied at the router group level -- no route accidentally left public.
- All queries via GORM parameterized. No raw SQL string concatenation.
- CORS restricted to `FRONTEND_ORIGIN` env var, not `*`.
- `hp_percent_lost` constrained 1-100 at both validation and DB check constraint level.

> Auth module (token issuance, refresh rotation, OAuth callback) warrants a `/security-review` before Phase 2 ships.

---

## 10. Risks

| Risk | Mitigation |
|---|---|
| Pokebase.app blocks scraper or changes page structure | Production DB is the durable snapshot -- app does not re-scrape at runtime. Fallback: free-text move entry. |
| Champions patch breaks learnset data | `champions-overrides.json` updated manually each patch. `cmd/seed` is re-runnable (upserts). |
| Turn input exceeds 20s on mobile | Validated on mid-range Android before hardening. Secondary effect prompts are first candidate to simplify. |
| Refresh token theft | Rotation on every refresh limits damage. 15-min access token TTL limits blast radius. |

---

## 11. Open Questions

| Question | Blocking |
|---|---|
| Hosting provider (Render, Railway, Fly.io, VPS) | Not blocking Phase 1-2 |
| `training_points` exact schema | Not blocking -- ship as free JSONB, tighten once Champions UI is confirmed |
| Refresh token: fixed 30-day TTL vs sliding window | Not blocking -- fixed TTL ships first |
| `our_picks` lead/back role: store in pick phase JSONB or derive from turn 1 | Blocks pick-phase endpoint design |

---

## 12. Related Docs

- `docs/PRD.md` -- what and why
- `docs/wireframes.md` -- UI layout
- `docs/design-system.md` -- color tokens, typography, animation, component conventions
- `docs/adr/001` -- PostgreSQL over MongoDB
- `docs/adr/002` -- GORM over sqlc/raw SQL
- `docs/adr/003` -- Go + Chi over NestJS
- `docs/adr/004` -- SvelteKit over React/Next.js
- `docs/adr/005` -- JWT + refresh rotation over server sessions
- `docs/adr/006` -- seed pipeline design
- `docs/adr/007` -- seed data visibility (gitignored JSON files)
- `docs/adr/008` -- localStorage persistence strategy
