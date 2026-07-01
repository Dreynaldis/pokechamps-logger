# Tech Spec: PokeChamps Logger -- Full System

**Status:** Draft  ·  **Author:** Dion  ·  **Date:** 2026-06-26

---

## 1. Summary

PokeChamps Logger is a PWA-compatible web app where authenticated users build Pokemon teams and log every turn of a Pokemon Champions match.
The system is a React SPA frontend backed by a NestJS REST API and a PostgreSQL database.
Reference data (Pokemon, moves, abilities, items, learnsets) is seeded from PokeAPI and pokebase.app, with a Champions-specific override layer maintained in the repo.
All match data is private per user; no public sharing in v1.

---

## 2. Problem, Goals & Non-Goals

**Problem:** No in-game replay system exists for Pokemon Champions; players lose all match detail the moment the game ends.

**Goals:**
- Team builder: save a team (6 Pokemon, min 1 move each) and mark one team as active.
- Battle logger: log pick phase, leads, and every turn action (move or switch) including HP% events and secondary effects, in under 20 seconds per turn on mobile.
- Match history: list view and turn-by-turn detail view for past matches.
- Autocomplete Pokemon/move search within 100ms per keystroke.
- Auth: email + password, Google OAuth, Discord OAuth.

**Non-goals (v1):**
- Damage calculation or stat computation (HP% is user-reported, not computed).
- Match sharing between users.
- Offline-first support (PWA shell loads fast but data requires network).
- Learnset validation if pokebase.app data is unavailable (fallback: free-text move entry).
- Tournament mode.

---

## 3. Proposed Design

### 3.1 SQL vs NoSQL

**PostgreSQL (relational) is the right choice.** Here is why:

The data is inherently relational at every layer:
- A Pokemon has 1-3 ability slots; each slot maps to an ability entity.
- A Pokemon can learn many moves; each move is a shared entity with its own stats (power, accuracy, pp, type).
- A match has many turns; a turn has many actions; an action has many HP events.
- Future stats queries (win rate per team, most-used leads) require aggregation across structured rows -- SQL is directly suited for this.

NoSQL would help if the data were schemaless or the read pattern were always "fetch the whole document." Neither is true here: the match list view and the match detail view need different projections, and the reference data has well-defined, stable structure.

**Where JSONB is used within PostgreSQL:**
- `pokemon.stats` -- exactly 6 fixed named stats; no need for a join table.
- `pokemon.types` -- max 2 type strings; a `text[]` column is sufficient.
- `team_slots.training_points` -- EV-equivalent data whose exact shape depends on what Champions exposes; JSONB lets us tighten the schema later without a migration.
- `secondary_effects.detail` -- the payload per secondary effect type is genuinely variable (a burn has a target, a weather change has a duration, a stat drop has a stat name and stage count). JSONB is the right fit.

### 3.2 Tech Stack

#### Backend: Go + Chi + GORM

Go is chosen because:
- The API is straightforward REST CRUD -- no complex framework machinery is needed, and Go's explicit, no-magic style is the right fit.
  A full DI framework (NestJS) would add ceremony without buying anything for a service this simple.
- Chi is a lightweight, idiomatic Go HTTP router built on top of the standard `net/http` package.
  Middleware (JWT auth, request logging, recovery) is composed explicitly as a chain -- nothing is hidden.
  This makes the auth guard visible and auditable at every route group, which is more learnable than NestJS decorators.
- Go's goroutine model handles concurrent writes (multiple turn actions per turn) efficiently.
  The I/O-bound write pattern suits Go just as well as Node.js, with lower memory overhead per request.
- Single binary deployment: `go build` produces one executable with no runtime dependency.
  No Node.js version to manage on the server.
- Statically typed with no runtime type coercion -- struct validation errors are caught at compile time, not at request time.

ORM: **GORM**.
GORM is the standard Go ORM.
Models are plain Go structs with field tags (similar role to TypeORM entities, different syntax).
Supports PostgreSQL, UUIDs, JSONB columns, and auto-migration.
Migration story is comparable to TypeORM migrations.

Auth libraries:
- `golang-jwt/jwt` -- JWT signing and verification.
- `markbates/goth` -- OAuth2 provider adapters for Google and Discord (same conceptual role as Passport.js strategies).
- `golang.org/x/crypto/bcrypt` -- password hashing at cost 12.
- `go-playground/validator/v10` -- struct tag-based request body validation (same role as class-validator + NestJS ValidationPipe).

**Type bridge -- Go backend to SvelteKit frontend:**
Go and TypeScript do not share a type system.
The bridge is an OpenAPI spec generated from Go annotations (`swaggo/swag`), consumed by `openapi-typescript` to generate TypeScript types for the frontend.
This codegen step runs as part of the build pipeline and keeps API response shapes in sync without manual duplication.

#### Frontend: SvelteKit (SPA mode)

SvelteKit + Svelte is chosen because:
- Svelte compiles components to plain JavaScript at build time -- no virtual DOM runtime is shipped to the browser.
  The resulting bundle is smaller and faster to parse on mobile, which is the primary logging device.
- Svelte's reactivity model suits the turn logger's complex local state.
  In React, a turn logger slot needs `useState` for action type, move selection, HP events array, and secondary effects -- each requiring explicit setter calls.
  In Svelte, `let actionType = 'move'` is reactive by default; derived values use `$:` labels.
  Less boilerplate for the most state-heavy screen in the app.
- SSR is disabled (`adapter-static` + `ssr: false`) because all pages are auth-gated.
  No SEO benefit from server rendering; disabling it removes hydration complexity.
- `vite-plugin-pwa` integrates the same way it would with a plain Vite project -- PWA support is not SvelteKit-specific.
- Svelte stores (writable, derived) replace React context for auth token state and active team state.
- TypeScript is supported natively via `<script lang="ts">` in `.svelte` files.

#### Database: PostgreSQL

- Standard choice for multi-user account-based systems.
- `pg_trgm` trigram index covers the 100ms autocomplete requirement on 310 Pokemon and a few hundred moves without a separate search service.
- JSONB available for the flexible columns described above.
- UUID primary keys via `gen_random_uuid()` built-in.
- Handles concurrent writes correctly (unlike SQLite).

### 3.3 Architecture

```
Browser (SvelteKit SPA + PWA manifest)
        |
        | HTTPS / JSON REST
        v
Go API  (Chi router -- auth, teams, matches, seed data read endpoints)
        |
        | GORM
        v
PostgreSQL
        ^
        |
Seed scripts (one-time, run at deploy)
  - cmd/scrape/main.go    → seed/pokebase-raw.json
  - cmd/enrich/main.go    → seed/moves-enriched.json, seed/abilities-enriched.json
  - seed/champions-overrides.json (manually maintained)
  - go run ./cmd/seed     (upserts all the above into the DB)

Codegen (run after any API change)
  - swaggo/swag generates openapi.json from Go handler annotations
  - npx openapi-typescript openapi.json -o src/lib/api-types.ts
    (TypeScript types consumed by the SvelteKit frontend)
```

The SvelteKit build is a static output (`adapter-static`) served from a CDN or the same host as the API.
No BFF or GraphQL -- REST is sufficient for this data shape.

---

### 3.4 Data Model

All primary keys: `uuid DEFAULT gen_random_uuid()`.
All timestamps: `timestamptz NOT NULL DEFAULT now()`.

---

#### 3.4.1 Reference / Seed Tables

These tables are populated once by the seed pipeline and updated when a Champions patch ships.
They are read-only at runtime (no user writes).

```sql
-- A Pokemon species (310 in Champions v1)
pokemon (
  id          uuid PK,
  dex_number  int  UNIQUE NOT NULL,
  name        text UNIQUE NOT NULL,         -- lowercase, hyphenated (matches PokeAPI convention)
  display_name text NOT NULL,               -- "Charizard" (title case, for UI)
  sprite_url  text,
  types       text[] NOT NULL,              -- e.g. ['fire', 'flying']  max 2 entries
  base_stats  jsonb NOT NULL                -- {hp, attack, defense, sp_atk, sp_def, speed}
  -- base stats at level 50 stored separately in usage_hints (post-MVP)
)

-- An ability entity (one record per ability name)
abilities (
  id                   uuid PK,
  name                 text UNIQUE NOT NULL,
  display_name         text NOT NULL,
  short_effect         text,                -- English short description from PokeAPI
  grants_immunity_type text                 -- e.g. 'ground' (Levitate), 'electric' (Volt Absorb)
                                            -- used by turn logger for proactive immunity prompts
)

-- Join table: which abilities a Pokemon can have and in which slot
-- Slot 1 = first regular ability, slot 2 = second regular ability, slot 3 = hidden ability
-- Not all Pokemon have a slot 2 or slot 3
pokemon_abilities (
  pokemon_id uuid NOT NULL REFERENCES pokemon(id),
  ability_id uuid NOT NULL REFERENCES abilities(id),
  slot       int  NOT NULL CHECK (slot IN (1, 2, 3)),
  is_hidden  boolean NOT NULL DEFAULT false,
  PRIMARY KEY (pokemon_id, slot)
)

-- A move entity
moves (
  id                   uuid PK,
  name                 text UNIQUE NOT NULL,
  display_name         text NOT NULL,
  type                 text NOT NULL,       -- 'fire', 'water', etc.
  category             text NOT NULL,       -- 'physical' | 'special' | 'status'
  power                int,                 -- null for status moves
  accuracy             int,                 -- null for moves that never miss (e.g. Swift)
  pp                   int NOT NULL,
  priority             int NOT NULL DEFAULT 0,  -- negative = last, positive = first (e.g. Quick Attack = +1)
  target               text NOT NULL,       -- 'selected-pokemon' | 'all-opponents' | 'user' | etc.
                                            -- 'all-opponents' = spread move (Heat Wave hits both)
  short_effect         text,               -- English description
  effect_chance        int,                -- % chance of secondary effect (e.g. 10 for Heat Wave burn)
  has_secondary_effect boolean NOT NULL DEFAULT false,  -- true if any secondary effect can occur
  is_pivot             boolean NOT NULL DEFAULT false   -- true for Flip Turn, Volt Switch, U-turn, Parting Shot
)

-- Join table: which moves a Pokemon can use in Champions (Champions-specific learnset)
pokemon_learnsets (
  pokemon_id uuid NOT NULL REFERENCES pokemon(id),
  move_id    uuid NOT NULL REFERENCES moves(id),
  PRIMARY KEY (pokemon_id, move_id)
)

-- Held items (mega stones + battle items)
items (
  id                    uuid PK,
  name                  text UNIQUE NOT NULL,
  display_name          text NOT NULL,
  category              text NOT NULL,     -- 'mega-stones' | 'held-items' | etc.
  short_effect          text,
  field_effect_duration int               -- extra turns granted by the item:
                                          -- Light Clay +3 (screens), Damp Rock +3 (rain), etc.
                                          -- null if item does not extend a field effect
)

-- Mega forms: derived from Pokemon + held item combination
-- e.g. Charizard + Charizardite Y = Mega Charizard Y
mega_forms (
  id              uuid PK,
  base_pokemon_id uuid NOT NULL REFERENCES pokemon(id),
  item_id         uuid NOT NULL REFERENCES items(id),
  form_name       text NOT NULL,          -- 'Mega Charizard Y'
  sprite_url      text,
  UNIQUE (base_pokemon_id, item_id)
)
```

---

#### 3.4.2 User & Auth Tables

```sql
users (
  id            uuid PK,
  email         text UNIQUE NOT NULL,
  password_hash text,         -- null for OAuth-only accounts (user has no local password)
  created_at    timestamptz,
  updated_at    timestamptz
)

-- One row per OAuth provider connection per user
oauth_accounts (
  id                  uuid PK,
  user_id             uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider            text NOT NULL,        -- 'google' | 'discord'
  provider_account_id text NOT NULL,        -- the ID from the provider (stable across logins)
  created_at          timestamptz,
  UNIQUE (provider, provider_account_id)
)

-- Refresh tokens stored as hashes, never plaintext
refresh_tokens (
  id         uuid PK,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL,     -- bcrypt hash of the raw 32-byte random token
  expires_at timestamptz NOT NULL,
  created_at timestamptz
)
```

---

#### 3.4.3 Team Tables

```sql
teams (
  id         uuid PK,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       text NOT NULL,
  is_active  boolean NOT NULL DEFAULT false,
  created_at timestamptz,
  updated_at timestamptz
)

-- One row per Pokemon slot in a team (max 6)
team_slots (
  id              uuid PK,
  team_id         uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  slot            int  NOT NULL CHECK (slot BETWEEN 1 AND 6),
  pokemon_id      uuid NOT NULL REFERENCES pokemon(id),
  ability_id      uuid REFERENCES abilities(id),     -- must be one of pokemon's valid abilities
  item_id         uuid REFERENCES items(id),
  nature          text,
  training_points jsonb,    -- EV-equivalent; shape TBD once Champions UI is confirmed
  created_at      timestamptz,
  UNIQUE (team_id, slot)
)

-- One row per move in a slot (max 4 moves per slot, min 1 required to save)
team_slot_moves (
  id           uuid PK,
  team_slot_id uuid NOT NULL REFERENCES team_slots(id) ON DELETE CASCADE,
  move_id      uuid NOT NULL REFERENCES moves(id),  -- must be in pokemon's learnset
  slot         int  NOT NULL CHECK (slot BETWEEN 1 AND 4),
  UNIQUE (team_slot_id, slot)
)
```

**Active team constraint:** enforced at the application layer, not at the DB level.
When a team is set active, a transaction sets all other teams for that user to `is_active = false` first, then sets the target team to `true`.
This avoids a complex partial unique index and keeps the logic readable.

---

#### 3.4.4 Match Tables

```sql
matches (
  id           uuid PK,
  user_id      uuid NOT NULL REFERENCES users(id),
  team_id      uuid NOT NULL REFERENCES teams(id),  -- snapshot of which team was active
  result       text CHECK (result IN ('win', 'loss')),  -- null until match ends; concede = 'loss'
  started_at   timestamptz,
  completed_at timestamptz     -- null while match is in progress
)

-- The pick phase: which 4 of our 6 we brought, and up to 6 enemy Pokemon
match_pick_phase (
  id         uuid PK,
  match_id   uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE UNIQUE,
  our_picks  jsonb NOT NULL
  -- [{team_slot_id: uuid, role: 'lead' | 'back'}]  -- exactly 4 entries; role set at turn 1 leads
)

-- Enemy Pokemon seen during the match (0-6 entries per match)
-- A row is created when we see the Pokemon (pick phase or switch-in)
match_enemy_pokemon (
  id                 uuid PK,
  match_id           uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  pokemon_id         uuid REFERENCES pokemon(id),    -- null if not yet identified
  position           int,           -- 1-6 in their team; null if unknown
  identified_at_turn int            -- turn number they were first seen (null if seen at pick phase)
)

turns (
  id          uuid PK,
  match_id    uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  turn_number int NOT NULL,
  created_at  timestamptz,
  UNIQUE (match_id, turn_number)
)

-- One row per action within a turn (up to 4: 2 per side)
-- An action is either a move OR a switch (mutually exclusive per slot per turn)
-- Pivot moves (Flip Turn, Volt Switch, Parting Shot, U-turn) are logged as a move;
-- the forced switch-in is recorded in secondary_effects with effect_type = 'pivot_switch_in'
turn_actions (
  id                         uuid PK,
  turn_id                    uuid NOT NULL REFERENCES turns(id) ON DELETE CASCADE,
  priority_order             int  NOT NULL,   -- 1-4: the order actions occurred within the turn
  side                       text NOT NULL CHECK (side IN ('us', 'enemy')),

  -- Actor: exactly one of these two must be non-null depending on side
  our_team_slot_id           uuid REFERENCES team_slots(id),
  enemy_pokemon_id           uuid REFERENCES match_enemy_pokemon(id),

  action_type                text NOT NULL CHECK (action_type IN ('move', 'switch')),
  move_id                    uuid REFERENCES moves(id),    -- null when action_type = 'switch'
  outcome                    text CHECK (outcome IN ('hit', 'miss', 'protected', 'immune')),
                                               -- null when action_type = 'switch'
  has_priority               boolean NOT NULL DEFAULT false,

  -- Switch target: set when action_type = 'switch'; null otherwise
  -- (for pivot moves the switch-in goes into secondary_effects instead)
  switch_in_our_slot_id      uuid REFERENCES team_slots(id),
  switch_in_enemy_pokemon_id uuid REFERENCES match_enemy_pokemon(id)
)

-- HP loss events within an action, in the order they occurred
-- Each source is a separate row: primary damage, recoil, contact ability damage, weather, etc.
hp_events (
  id                      uuid PK,
  turn_action_id          uuid NOT NULL REFERENCES turn_actions(id) ON DELETE CASCADE,
  event_order             int  NOT NULL,  -- 1, 2, 3... within this action

  -- Target: exactly one of these two must be non-null
  target_our_slot_id      uuid REFERENCES team_slots(id),
  target_enemy_pokemon_id uuid REFERENCES match_enemy_pokemon(id),

  hp_percent_lost         int NOT NULL CHECK (hp_percent_lost BETWEEN 1 AND 100)
)

-- Secondary effects that occurred during an action
-- Shown only when the move/ability/item is flagged has_secondary_effect = true
secondary_effects (
  id             uuid PK,
  turn_action_id uuid NOT NULL REFERENCES turn_actions(id) ON DELETE CASCADE,
  effect_type    text NOT NULL,
  -- valid values:
  --   'burn' | 'freeze' | 'paralysis' | 'sleep' | 'poison' | 'bad_poison'
  --   'flinch' | 'confusion'
  --   'stat_change'        -- detail: {stat, stages, target_side, target_pokemon_id}
  --   'weather'            -- detail: {type: 'rain'|'sun'|'sand'|'hail', duration_turns}
  --   'field_effect'       -- detail: {type: 'trick_room'|'tailwind'|'reflect'|..., duration_turns, side}
  --   'mega_evolution'     -- detail: {side, actor_ref, form_name}
  --   'pivot_switch_in'    -- detail: {switch_in_our_slot_id?  switch_in_enemy_pokemon_id?}
  detail jsonb
)
```

---

### 3.5 Seed Data Pipeline

The reference tables are populated from two external sources merged with a Champions override layer.
The pipeline runs offline and outputs JSON files to `seed/`.
These files are gitignored -- they are not committed to the repo.
Anyone deploying the app runs the pipeline once to populate their local seed files before loading the DB.
The production DB is the stable snapshot; the JSON files are disposable intermediate artifacts.

```
Step 1 -- Scrape pokebase.app           → seed/pokebase-raw.json
  go run ./cmd/scrape
  For each of the 310 Champions Pokemon:
    - name, types, base stats
    - abilities list (name + description)
    - move list (name, type, category, power, accuracy, pp)
  Rate-limited to 1 req/sec. Output is gitignored (not committed to the repo).

Step 2 -- Enrich moves from PokeAPI      → seed/moves-enriched.json
  go run ./cmd/enrich --target=moves
  For each move name seen in pokebase-raw.json:
    - GET https://pokeapi.co/api/v2/move/{name}
    - Pull: priority, target, effect_chance, meta.ailment, meta.flinch_chance,
            meta.drain, meta.healing, stat_changes, effect_entries (English short_effect)
    - Derive has_secondary_effect:
        true if effect_chance > 0 OR meta.ailment != 'none' OR flinch_chance > 0
             OR stat_changes non-empty OR drain != 0 OR healing != 0
  PokeAPI is the authoritative source for these fields; pokebase does not expose them.

Step 3 -- Enrich abilities from PokeAPI  → seed/abilities-enriched.json
  go run ./cmd/enrich --target=abilities
  For each ability name seen in pokebase-raw.json:
    - GET https://pokeapi.co/api/v2/ability/{name}
    - Pull: English short_effect
    - Manually set grants_immunity_type in champions-overrides.json
      (PokeAPI does not expose immunity type as a discrete field)

Step 4 -- Apply Champions override layer  seed/champions-overrides.json
  Manually maintained JSON file updated each patch cycle (~monthly).
  Structure:
  {
    "move_patches": [
      { "name": "flip-turn", "patch": { "is_pivot": true } },
      { "name": "volt-switch", "patch": { "is_pivot": true } }
    ],
    "learnset_additions": [
      { "pokemon": "rillaboom", "moves": ["grassy-glide"] }
    ],
    "learnset_removals": [],
    "ability_patches": [
      { "name": "levitate", "patch": { "grants_immunity_type": "ground" } }
    ],
    "mega_forms": [
      {
        "base_pokemon": "charizard",
        "item": "charizardite-y",
        "form_name": "Mega Charizard Y",
        "sprite_url": "..."
      }
    ],
    "items": [
      { "name": "light-clay", "patch": { "field_effect_duration": 3 } }
    ]
  }

Step 5 -- Load into DB
  go run ./cmd/seed
  Runs a transaction that upserts all entities in dependency order:
  abilities → items → pokemon → moves → pokemon_abilities →
  pokemon_learnsets → mega_forms
  Re-runnable safely (upsert on unique name constraints).
```

**Why not use PokeAPI as the sole source?**
PokeAPI covers Gen 1-9 but Pokemon Champions has a specific 310-Pokemon roster and its own learnsets that may diverge from mainline games.
Pokebase.app is the only publicly available source of Champions-specific data.
PokeAPI fills in the fields pokebase does not expose (priority, target, meta data for secondary effects).

---

### 3.6 API Contracts

Base path: `/api/v1`.
All endpoints require `Authorization: Bearer <access_token>` unless marked public.

#### Auth

| Method | Path | Body | Notes |
|---|---|---|---|
| POST | `/auth/register` | `{email, password}` | Public. Returns `{user, accessToken}` + refresh cookie |
| POST | `/auth/login` | `{email, password}` | Public. Returns `{user, accessToken}` + refresh cookie |
| POST | `/auth/refresh` | -- | Reads refresh cookie, rotates it, returns new `{accessToken}` |
| POST | `/auth/logout` | -- | Deletes refresh token record. Returns 204 |
| GET | `/auth/google` | -- | Public. Starts Google OAuth redirect flow |
| GET | `/auth/google/callback` | -- | Public. Exchanges code, redirects to frontend |
| GET | `/auth/discord` | -- | Public |
| GET | `/auth/discord/callback` | -- | Public |
| GET | `/auth/me` | -- | Returns current user |

#### Teams

| Method | Path | Body | Notes |
|---|---|---|---|
| GET | `/teams` | -- | Returns all teams for current user, each with full slots + moves |
| POST | `/teams` | `{name}` | Creates empty team |
| PATCH | `/teams/:id` | `{name?, slots?}` | Upserts name and/or all slot data in one request |
| DELETE | `/teams/:id` | -- | 204 |
| POST | `/teams/:id/activate` | -- | Sets this team active; deactivates all others in a transaction |

`PATCH /teams/:id` `slots` body shape:
```json
[
  {
    "slot": 1,
    "pokemon_id": "uuid",
    "ability_id": "uuid",
    "item_id": "uuid",
    "nature": "timid",
    "training_points": {},
    "moves": [
      { "slot": 1, "move_id": "uuid" },
      { "slot": 2, "move_id": "uuid" }
    ]
  }
]
```
Server validates: 6 slots, each with at least 1 move, ability belongs to that Pokemon, moves belong to that Pokemon's learnset.

#### Matches

| Method | Path | Body | Notes |
|---|---|---|---|
| GET | `/matches` | `?page&limit` | Paginated. Returns list with team summary, enemy summary, result |
| POST | `/matches` | -- | Creates match using current active team. Fails if no active team |
| GET | `/matches/:id` | -- | Full detail: pick phase, all turns, all actions, HP events, secondary effects |
| POST | `/matches/:id/pick-phase` | `{our_picks, enemy_pokemon?}` | Sets the pick phase |
| POST | `/matches/:id/turns` | `{actions: [...]}` | Logs one full turn |
| POST | `/matches/:id/end` | `{result: 'win'|'loss'}` | Closes the match |

Turn action body shape:
```json
{
  "actions": [
    {
      "priority_order": 1,
      "side": "us",
      "our_team_slot_id": "uuid",
      "action_type": "move",
      "move_id": "uuid",
      "outcome": "hit",
      "has_priority": false,
      "hp_events": [
        {
          "event_order": 1,
          "target_enemy_pokemon_id": "uuid",
          "hp_percent_lost": 45
        },
        {
          "event_order": 2,
          "target_enemy_pokemon_id": "uuid-2",
          "hp_percent_lost": 38
        }
      ],
      "secondary_effects": [
        {
          "effect_type": "burn",
          "detail": { "target_enemy_pokemon_id": "uuid" }
        }
      ]
    },
    {
      "priority_order": 2,
      "side": "enemy",
      "enemy_pokemon_id": "uuid",
      "action_type": "switch",
      "switch_in_enemy_pokemon_id": "uuid-new"
    }
  ]
}
```

#### Reference Data (read-only)

| Method | Path | Query | Notes |
|---|---|---|---|
| GET | `/pokemon` | `?q=char&limit=10` | Trigram search on display_name. Used for enemy pick entry |
| GET | `/pokemon/:id` | -- | Full Pokemon: abilities, learnset with move stats, mega forms |
| GET | `/moves` | `?q=heat&pokemon_id=uuid` | Move search, filtered to learnset if pokemon_id provided |
| GET | `/items` | `?q=char` | Item search for team builder |

---

### 3.7 Auth Flow

**Email/password:**
1. Registration: bcrypt-hash password at cost 12, insert `users` row.
2. Login: compare hash, issue tokens on match.
3. Access token: JWT, signed with `AUTH_SECRET` env var, 15-minute expiry. Contains `{sub: userId}`.
4. Refresh token: 32 random bytes, bcrypt-hashed and stored in `refresh_tokens`. Sent as `HttpOnly; Secure; SameSite=Strict` cookie. 30-day expiry.
5. Refresh: read cookie, find matching hash in DB, delete old row, issue new access token and new refresh token (rotation).
6. Logout: delete refresh token row. Client drops access token from memory.

**OAuth (Google / Discord):**
1. `/auth/google` redirects to provider consent screen with `state` param (CSRF protection).
2. Callback validates `state`, exchanges code for profile via Goth provider adapter.
3. Find `oauth_accounts` row by `(provider, provider_account_id)`. If found, load linked user. If not found, create `users` row + `oauth_accounts` row.
4. Issue same JWT + refresh cookie as email flow.
5. Redirect to `/dashboard#token=<accessToken>`. SPA reads from hash, stores in a Svelte writable store (memory only), clears the hash from the URL.

**Client token storage:** access token lives in a Svelte writable store (memory only). Refresh token in HttpOnly cookie. A fetch wrapper catches 401 responses, calls `/auth/refresh`, retries the original request.

---

## 4. Alternatives Considered

**MongoDB / document store instead of PostgreSQL**
Considered because match detail is a deeply nested document (match → turns → actions → events).
Rejected because the nested structure is consistent and well-defined (not schemaless), future stats queries need aggregation across structured rows (SQL is directly suited), and using JSONB for the genuinely flexible parts (secondary_effects.detail, training_points) gives the benefits of document storage without giving up relational integrity.

**NestJS (TypeScript/Node.js) instead of Go + Chi**
Considered because the developer already knows TypeScript and NestJS has built-in DI, guards, and pipes.
Rejected because: the API is simple CRUD -- NestJS's framework machinery buys nothing at this scope; Go produces a single static binary with simpler deployment; and using Go is a deliberate learning goal.
The type-sharing benefit of TypeScript end-to-end is replaced by the OpenAPI codegen bridge (swaggo → openapi-typescript), which is an explicit part of the build pipeline.

**React + Vite instead of SvelteKit**
Rejected because: Svelte compiles away its runtime, producing a smaller bundle for mobile-first use; the turn logger's complex local state is cleaner to express in Svelte's native reactivity than with React hooks; and Svelte is a deliberate learning goal.
SvelteKit in SPA mode (SSR disabled) covers PWA, routing, and TypeScript with the same vite-plugin-pwa integration.

**GraphQL instead of REST**
Considered because match detail has a deeply nested response.
Rejected because the query graph is fixed and small -- the API has no consumers other than the first-party SPA, so the flexibility of GraphQL adds no value and increases the learning surface.

**sqlc or raw SQL instead of GORM**
sqlc generates type-safe Go functions from hand-written SQL -- zero runtime overhead, no magic.
Deferred because GORM's auto-migration is useful during early schema iteration.
Revisit after Phase 3 if GORM's query builder becomes a bottleneck.

**Next.js instead of SvelteKit**
SSR and file-based routing.
Rejected because all pages are auth-gated (no SEO benefit), data is user-specific (no server-render benefit), and the hydration complexity would conflict with the real-time-ish turn logger UI.

**Separate search service (Typesense / Elasticsearch) for autocomplete**
Overkill for 310 Pokemon and a few hundred moves.
PostgreSQL `pg_trgm` with a GIN index delivers well under 100ms on a dataset this size with zero additional infrastructure.

**Storing all Gen 1-9 move data (all ~900 moves from PokeAPI) instead of filtering**
Rejected because the team builder must only show moves that Champions Pokemon can learn.
A full dump would mean filtering at query time on every autocomplete keystroke.
Seeding only the moves that appear in at least one Champions Pokemon learnset keeps the move table small (a few hundred rows) and the learnset join table efficient.

---

## 5. Security & Privacy

**Authentication:**
- Passwords hashed with bcrypt cost 12. Never logged, never returned in any response.
- Refresh tokens stored as bcrypt hashes. The raw token travels only in an HttpOnly cookie -- JavaScript cannot read it.
- Access token signed with `AUTH_SECRET` from env var. Secret is never committed to the repo.
- OAuth `state` parameter validated on callback to prevent CSRF.

**Authorization:**
- Every data endpoint scopes queries to `WHERE user_id = <authenticated user id>`. No endpoint allows cross-user data access.
- JWT auth middleware applied to the Chi router group that covers all protected routes. No endpoint is accidentally left public.
- `team_id` and `match_id` path params are verified to belong to the current user before any read or write.

**Input validation:**
- `go-playground/validator/v10` with struct tags on all request body structs.
- `hp_percent_lost` constrained 1-100 at both struct validation and DB check constraint level.
- `move_id`, `pokemon_id`, `item_id`, `ability_id` in user-submitted bodies validated against the seed tables via FK (DB) and an existence check (application layer returns 422 not 500 on FK violation).
- Free-text move entry fallback (if learnset data unavailable): max 100 chars, HTML-escaped at the handler layer.

**Injection:**
- GORM parameterizes all queries. No raw SQL string concatenation.
- Trigram search uses GORM's raw query with positional parameters for the `similarity()` call.

**Data privacy:**
- Match data scoped to `user_id` -- no public endpoint exposes another user's match.
- Email is not logged. No PII beyond email is stored.
- CORS restricted to the frontend origin via env var, not `*`.

**Secrets:**
- `AUTH_SECRET`, `DATABASE_URL`, `GOOGLE_CLIENT_SECRET`, `DISCORD_CLIENT_SECRET` via env vars.
- `.env` is `.gitignore`d. `.env.example` with placeholder values is committed.

> The auth module (token issuance, refresh rotation, OAuth callback) warrants a `/security-review` pass before it is considered done.

---

## 6. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Pokebase.app blocks the scraper or changes page structure | High -- learnset data unavailable | The production DB is the durable snapshot -- once seeded, the app does not re-scrape at runtime. Seed JSON files are gitignored but can be regenerated by re-running the scraper. Fallback if site structure breaks: free-text move entry in the team builder. |
| Champions patch breaks learnset or move data | Medium -- team builder shows stale data | `champions-overrides.json` is updated manually each patch cycle. `db:seed` is re-runnable. A stale move entry is cosmetic only; it does not corrupt match logs (matches store IDs, not names). |
| Turn input exceeds 20-second target on mobile | High -- core UX failure | Validated with real players before hardening. Secondary effect prompts are the first candidate to simplify if too slow. Autocomplete tested on mid-range Android hardware. |
| Refresh token theft (stolen cookie) | High -- account takeover | Rotation on every refresh (stolen token can only be used once before invalidated). Short access token TTL (15 min) limits blast radius of a stolen access token. |
| `pg_trgm` too slow for autocomplete | Low | GIN trigram index handles thousands of rows under 20ms. 310 Pokemon is a non-issue. Only a risk if the Pokemon table grew to tens of thousands, which is not planned. |

---

## 7. Rollout Plan

No existing users or data to migrate. Greenfield build.

**Phase 0 -- Project scaffold**
- Go module init, SvelteKit init, Docker Compose for PostgreSQL, DB connection, first GORM migration.
- Exit: `go run ./cmd/api` connects to Postgres and returns 200 on `/health`. Migrations run clean.

**Phase 1 -- Seed pipeline + data verification**
- Run scraper (`go run ./cmd/scrape`), enrich from PokeAPI (`go run ./cmd/enrich`), apply overrides, load DB (`go run ./cmd/seed`).
- Exit: all 310 Pokemon queryable via a seed-check script. Spot-check learnsets and move metadata for 5+ Pokemon. Credits pokebase.app in the app footer.

**Phase 2 -- Auth (backend only)**
- Email/password register + login, JWT + refresh token rotation, Google + Discord OAuth via Goth, `/auth/me`.
- Exit: full auth cycle exercised via curl. OAuth callback tested manually. Cross-user isolation verified (user A's token rejected on user B's resources).

**Phase 3 -- Team builder (backend + frontend)**
- Team CRUD endpoints + active team selection. SvelteKit team builder UI: create team, fill 6 Pokemon slots with ability/item/moves, set active team.
- Exit: create 2+ teams in the UI, switch active team, verify correct team returned as active. Minimum-move validation enforced in the UI and rejected at the API.

**Phase 4 -- Match logging (backend + frontend)**
- Match creation, pick phase, turn-by-turn logging (move/switch, HP events, secondary effects), match-end.
- SvelteKit screens: pick phase, turn logger.
- Exit: log a full 3-turn match with at least 1 HP event, 1 secondary effect, and 1 switch action. Full match retrievable via API with correct ordering.

**Phase 5 -- Match history (backend + frontend)**
- Paginated match list API + UI. Match detail view (pick phase + turn-by-turn replay).
- Exit: completed match appears in history list with correct team/result summary. Detail view shows all actions in correct priority order.

**Phase 6 -- PWA + mobile hardening**
- PWA manifest + service worker via vite-plugin-pwa. Test on a real mid-range Android device.
- Exit: installable as PWA. Full turn logging under 20 seconds on target device. No regressions from Phase 4 manual test.

No feature flags needed -- no existing user base to protect.
Future schema changes use GORM migrations (`go run ./cmd/migrate`).

---

## 8. Testing Strategy

**Backend -- unit tests (Go `testing` package):**
- Active team constraint: setting team B active deactivates team A in the same user scope; another user's teams are unaffected.
- Turn action validation: move and switch mutually exclusive per slot; pivot move creates a `pivot_switch_in` secondary effect not a switch action.
- HP event ordering: events inserted with explicit `event_order` values, returned in order.
- Pick phase validation: exactly 4 picks required; picks must be from the active team's slots.

**Backend -- integration tests (Go `testing` + real PostgreSQL via Docker):**
- Auth: register → login → refresh → logout cycle. OAuth account linking. Duplicate email rejection.
- Full match lifecycle: create match → pick phase → 2 turns with HP events → end match → fetch detail.
- Cross-user isolation: user A cannot read user B's matches or teams (expect 403/404).

**Frontend -- component tests (Vitest + Svelte Testing Library):**
- Autocomplete: debounce timing (100ms), keyboard navigation, selection.
- Turn logger slot: selecting Move disables Switch and vice versa. Cannot submit a turn with unfilled active slots.

**Manual validation (before each phase ships):**
- Tested on a mid-range Android device in a real browser session.
- Phase 2 done: register, log in via Google/Discord.
- Phase 3 done: build a 6-Pokemon team with moves, set active, verify active team in UI.
- Phase 4 done: start match, log 3 turns with at least one HP event and one secondary effect, end match.
- Phase 5 done: match appears in history list; detail view shows correct turn order and all logged events.
- Phase 6 done: PWA installable on Android, full turn logged in under 20 seconds.

---

## 9. Open Questions

| Question | Blocking | Notes |
|---|---|---|
| Hosting provider (Render, Railway, Fly.io, VPS) | Not blocking Phase 1-2 | Needed before any deploy. All options support PostgreSQL + Node.js containers. |
| `training_points` exact shape | Not blocking | Champions UI exposes EV-like values but the exact field names are unconfirmed. Ship as free JSONB and tighten once confirmed. |
| Ability slot data from pokebase.app | Blocks ability dropdown in team builder | During scrape: check whether pokebase exposes ability slot 1 / slot 2 / hidden distinction. If not, fall back to PokeAPI `pokemon/{name}` ability list. |
| Refresh token TTL: sliding window vs fixed 30-day | Not blocking | Fixed 30-day is simpler to implement first. Sliding window (extend on each use) is a follow-up if users complain about being logged out. |
| Should `our_picks` in `match_pick_phase` store lead/back roles, or derive them from turn 1 leads? | Blocks pick phase API design | Pick phase is set before turn 1. The role ('lead'/'back') within the 4 picks is confirmed at turn 1 lead selection. One option: store roles in `our_picks` JSONB and update them when turn 1 is logged. Other option: store roles separately in turn 1 data only. Needs a decision before implementing the pick-phase endpoint. |

---

## 10. Related Docs

- **Up (what & why):** `docs/PRD.md`
- **Up (UI layout):** `docs/wireframes.md`
- **Down (decisions to be written as ADRs):**
  - Database: PostgreSQL over MongoDB
  - ORM: TypeORM over Prisma
  - Frontend: React + Vite over Next.js
  - Auth: JWT + refresh rotation over server-side sessions
  - Seed strategy: pokebase + PokeAPI + override layer
