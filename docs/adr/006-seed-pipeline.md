# ADR 006: Seed Pipeline -- Dual Source + Manual Override Layer

**Status:** Accepted   ·   **Date:** 2026-06-27

## Context

The app requires reference data for 310 Pokemon (species, types, base stats, abilities, learnsets) and their learnable moves (power, accuracy, PP, priority, target, secondary effect flags).
No single publicly available source covers all required fields for the Champions-specific roster:
- PokeAPI covers Gen 1-9 moves comprehensively (priority, target, secondary effect metadata) but its Pokemon roster and learnsets reflect mainline games, which diverge from Champions.
- Pokebase.app covers the Champions-specific 310-Pokemon roster and learnsets but does not expose move priority, target, or secondary effect metadata.
- Champions patches ship roughly monthly and may change learnsets or move availability.

## Decision

Use a three-source pipeline:

1. **Scrape pokebase.app** (`go run ./cmd/scrape`) for the Champions-specific Pokemon roster, learnsets, abilities, and base move stats. Rate-limited to 1 req/sec. Output: `seed/pokebase-raw.json`.
2. **Enrich from PokeAPI** (`go run ./cmd/enrich`) for move priority, target, secondary effect flags, and ability descriptions. Output: `seed/moves-enriched.json`, `seed/abilities-enriched.json`.
3. **Apply a manual override layer** (`seed/champions-overrides.json`, committed to the repo) for Champion-specific corrections: pivot move flags, learnset additions/removals, ability immunity types, mega form definitions, item field-effect durations.
4. **Load into DB** (`go run ./cmd/seed`) upserts all entities in dependency order. Re-runnable safely.

Credit pokebase.app in the app footer.

## Options Considered

### Option A -- PokeAPI as sole source
- Pros: single source, well-maintained, no scraping required; explicit API with stable endpoints.
- Cons: PokeAPI does not have the Champions-specific 310-Pokemon roster or learnsets; learnsets in mainline games differ from Champions; Champions-specific abilities or item interactions are not reflected in PokeAPI.

### Option B -- Pokebase.app as sole source
- Pros: Champions-specific data for the correct roster.
- Cons: does not expose move priority, target, or secondary effect metadata -- these are required by the turn logger to prompt for secondary effects at the right time and to sort actions by priority order.

### Option C -- Dual source + manual override (chosen)
- Pros: complete data coverage; override layer handles Champions-specific corrections and patch-cycle updates without touching the scraper or enrichment scripts; pipeline is re-runnable and idempotent.
- Cons: dependency on pokebase.app page structure -- if it changes, the scraper breaks; manual override maintenance is ongoing work per patch cycle; two external HTTP dependencies instead of one.

## Consequences

- **Positive:** complete and accurate reference data. The override layer allows patch-cycle updates without re-engineering the pipeline. PokeAPI data is stable and well-documented.
- **Negative:** if pokebase.app changes its HTML structure, the scraper must be updated before a reseed is possible. The manual override file must be reviewed and updated after every Champions patch. A scraper failure does not affect the running app (the DB is already seeded) but blocks the next reseed.

## Fallback

If learnset data is unavailable (scraper broken, pokebase down), the team builder falls back to free-text move entry (max 100 chars). Match logs are unaffected -- they store entity IDs, not names.

## Related Docs

- Tech spec section 3.5 (seed pipeline): `docs/tech-spec.md`
- Tech spec section 6 (risks -- scraper failure): `docs/tech-spec.md`
- ADR 007 (seed data visibility): `docs/adr/007-seed-data-visibility.md`
