# ADR 007: Seed Data -- Gitignored JSON Files, Production DB as Durable Snapshot

**Status:** Accepted   ·   **Date:** 2026-06-27

## Context

The seed pipeline produces intermediate JSON files (`seed/pokebase-raw.json`, `seed/moves-enriched.json`, `seed/abilities-enriched.json`) by scraping pokebase.app and enriching from PokeAPI.
The repo is public (portfolio visibility).
Committing scraped data to a public repo re-distributes pokebase.app's content -- anyone who forks the repo gets the raw data without visiting their site, bypassing their traffic and any user engagement.
The scraper code itself (showing the data pipeline approach) is a legitimate portfolio artifact; the extracted data files are not.

## Decision

Add `seed/*.json` to `.gitignore`.
The scraper scripts (`cmd/scrape/`, `cmd/enrich/`, `cmd/seed/`) are committed and publicly visible.
`seed/champions-overrides.json` is committed (it is manually authored content, not scraped data).
The production database is the durable snapshot -- once seeded, the app does not depend on the JSON files at runtime.
Anyone deploying the app runs the seed pipeline once against their own environment.

## Options Considered

### Option A -- Commit seed JSONs to the repo
- Pros: anyone can clone and seed without running the scraper; acts as a stable snapshot if pokebase goes down.
- Cons: redistributes pokebase.app's data publicly -- ethically problematic even with attribution; the repo permanently hosts a copy of their content; if scraped data contains errors, it is harder to correct (committed files vs. re-running a script).

### Option B -- Gitignore seed JSONs, production DB as snapshot (chosen)
- Pros: no data redistribution; scraper code is public and demonstrates the pipeline approach; production DB persists seeded data -- existing deployments are unaffected if pokebase goes down; re-seeding for a new deployment requires running the scraper, which is the correct and honest path.
- Cons: a new deployment cannot seed if pokebase.app is down or has changed its structure; deployers must run the pipeline before the app is usable.

### Option C -- Separate private repo for scraper + seed data
- Pros: cleanest separation -- scraper code and data are entirely private; app repo is fully clean.
- Cons: two repos to manage; the scraper pipeline as a portfolio artifact is hidden; overhead with no meaningful benefit over Option B given the repo is already public.

## Consequences

- **Positive:** the public repo shows the scraping and data-pipeline approach (portfolio value) without redistributing third-party data. Production DB is self-contained -- no runtime dependency on the JSON files. `champions-overrides.json` is committed because it is original authored content, not extracted data.
- **Negative:** a fresh deployment requires internet access to pokebase.app and PokeAPI at seed time. If pokebase.app breaks or disappears, new deployments cannot be seeded until the scraper is updated or an alternative data source is found.

## Attribution

Pokebase.app is credited in the app footer on every page.
The README documents that Champions-specific data is sourced from pokebase.app with permission-free rate-limited scraping.

## Related Docs

- Tech spec section 3.5 (seed pipeline): `docs/tech-spec.md`
- Tech spec section 6 (risks): `docs/tech-spec.md`
- ADR 006 (seed pipeline strategy): `docs/adr/006-seed-pipeline.md`
