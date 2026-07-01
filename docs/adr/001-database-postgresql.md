# ADR 001: PostgreSQL over MongoDB

**Status:** Accepted   ·   **Date:** 2026-06-27

## Context

The app stores user accounts, teams (structured, FK-linked entities), and match logs (match → turns → actions → HP events → secondary effects).
The match log structure is deeply nested but consistent and well-defined -- it is not schemaless.
Future post-MVP features include win-rate aggregations and most-used leads, which require aggregation across structured rows.
Some columns are genuinely variable in shape: `secondary_effects.detail` varies per effect type, and `training_points` depends on what the Champions UI exposes.

## Decision

Use PostgreSQL as the sole database.
Use JSONB columns only for the three columns where the schema is genuinely variable (`training_points`, `secondary_effects.detail`, `pokemon.stats`/`pokemon.types`).
All other data is fully relational.

## Options Considered

### Option A -- PostgreSQL (relational + selective JSONB)
- Pros: relational integrity enforced at the DB level via FKs; SQL aggregation directly suits future stats queries; JSONB available for the flexible columns; `pg_trgm` GIN index covers the 100ms autocomplete requirement with no extra infrastructure; concurrent writes handled correctly (unlike SQLite); standard choice for multi-user account-based systems.
- Cons: JSONB columns lose schema enforcement -- a bad write to `secondary_effects.detail` is not caught by the DB; requires a migration for any structural change to relational columns.

### Option B -- MongoDB (document store)
- Pros: the match log nests naturally as a single document; no join overhead when fetching a full match detail.
- Cons: the nested structure is consistent and well-defined, so the main benefit of MongoDB (schemaless flexibility) does not apply here; future stats aggregations (win rate, most-used leads) are significantly harder to express in MongoDB's aggregation pipeline than in SQL; relational integrity between users, teams, and matches would be enforced only at the application layer.

## Consequences

- **Positive:** relational integrity is enforced at the DB level. Future aggregation queries are straightforward SQL. `pg_trgm` handles autocomplete with zero additional infrastructure.
- **Negative:** `secondary_effects.detail` and `training_points` are JSONB -- their shape must be validated at the application layer, not the DB. Schema changes to relational columns require migrations.

## Related Docs

- Tech spec section 3.1 (SQL vs NoSQL): `docs/tech-spec.md`
- Tech spec section 3.4 (data model): `docs/tech-spec.md`
