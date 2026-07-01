# PokeChamps Logger

A web app for logging and reviewing Pokemon Champions matches.
Track your battles turn by turn, review your team performance, and spot patterns across your match history.

> **Status:** Active development -- Phase 1 (seed pipeline + data verification) complete.
> Phase 2 (auth) is next.

## What it does

- **Team builder** -- register your 6-Pokemon team with moves, items, and abilities before a match.
- **Battle logger** -- log each turn's actions (moves, switches, HP events) in real time on mobile.
- **Match history** -- review past matches turn by turn; identify leads, damage sequences, and outcomes.

Built specifically for the [Pokemon Champions](https://pokebase.app) format: doubles (2v2), 6 registered, bring 4.

## Tech stack

| Layer | Tech |
|---|---|
| Backend | Go + [Chi](https://github.com/go-chi/chi) + [GORM](https://gorm.io) |
| Frontend | [SvelteKit](https://kit.svelte.dev) (SPA mode, SSR disabled) + Vite PWA |
| Database | PostgreSQL (JSONB for flexible columns) |
| Auth | JWT (15 min) + refresh token rotation (30 days, HttpOnly cookie) + Google/Discord OAuth |

## Project structure

```
cmd/
  api/      -- HTTP server (Chi router, GORM, JWT auth)
  scrape/   -- one-shot: scrapes pokebase.app roster into seed/pokebase-raw.json
  enrich/   -- one-shot: fetches PokeAPI metadata into seed/*-enriched.json
  seed/     -- one-shot: upserts all seed data into PostgreSQL
internal/
  config/   -- environment config (.env)
  database/ -- Postgres connection + AutoMigrate
  handler/  -- HTTP handler functions
  model/    -- GORM models (Pokemon, Move, Ability, learnset join tables)
frontend/   -- SvelteKit app
seed/
  champions-overrides.json  -- hand-curated patch overrides (committed)
  *.json                    -- generated seed files (gitignored, regenerate via cmd/)
docs/
  PRD.md, tech-spec.md, wireframes.md, adr/
```

## Local setup

**Prerequisites:** Go 1.22+, Node 20+, Docker

```bash
# 1. Start Postgres
docker compose up -d

# 2. Copy and fill in env
cp .env.example .env

# 3. Run the seed pipeline (one time)
go run ./cmd/scrape   # scrapes pokebase.app -> seed/pokebase-raw.json
go run ./cmd/enrich   # hits PokeAPI        -> seed/*-enriched.json
go run ./cmd/seed     # loads DB

# 4. Start the API server
go run ./cmd/api

# 5. Start the frontend (separate terminal)
cd frontend && npm install && npm run dev
```

API runs on `http://localhost:8080`, frontend on `http://localhost:5173`.

## Phases

| Phase | Description | Status |
|---|---|---|
| 0 | Repo scaffold, DB connection, `/health` | Done |
| 1 | Seed pipeline, data verification UI | Done |
| 2 | Auth (email + Google + Discord OAuth) | Next |
| 3 | Team builder (CRUD + UI) | Planned |
| 4 | Match logger (pick phase + turn UI) | Planned |
| 5 | Match history (list + detail views) | Planned |
| 6 | PWA hardening, mobile performance | Planned |

## Data sources

- **Pokemon roster, learnsets, move stats** -- scraped from [pokebase.app](https://pokebase.app) at 1 req/sec.
  pokebase.app is the community resource for Pokemon Champions data.
  All credit for the game data goes to the pokebase.app team.
- **Move metadata** (priority, target, secondary effects) -- [PokeAPI](https://pokeapi.co) (free, open).
- **Champions overrides** -- `seed/champions-overrides.json`, maintained manually each patch cycle.
