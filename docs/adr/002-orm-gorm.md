# ADR 002: ORM -- GORM over sqlc or raw SQL

**Status:** Accepted   ·   **Date:** 2026-06-27

## Context

The backend is written in Go.
Go does not have a built-in ORM.
The data model is actively evolving during early development (Phases 0-3) -- schema changes will be frequent.
The queries themselves are straightforward CRUD with one non-trivial case: the trigram similarity search for autocomplete.

## Decision

Use GORM as the ORM for the initial build.
Revisit after Phase 3 if GORM's query builder becomes a bottleneck or its magic causes debugging pain.

## Options Considered

### Option A -- GORM
- Pros: auto-migration is useful during active schema iteration; model structs with field tags map directly to DB columns; associations (has-many, belongs-to) reduce boilerplate for the team/match entity graph; widely used in the Go ecosystem with good PostgreSQL support including JSONB and UUID.
- Cons: runtime magic -- GORM infers queries from struct tags, which can produce surprising SQL (N+1 on eager-loaded associations if not careful); slightly less idiomatic Go than raw `database/sql`; type safety is weaker than sqlc (GORM returns `interface{}` in some paths).

### Option B -- sqlc
- Pros: generates fully type-safe Go functions from hand-written SQL at compile time; zero runtime overhead; the generated code is readable and auditable; forces explicit SQL, which is educational and leaves no hidden query behavior.
- Cons: requires writing raw SQL for every query, including migrations; during early schema iteration this is slower (change schema → rewrite SQL → regenerate); the compile-time codegen step adds tooling setup overhead before the first line of app code runs.

### Option C -- raw `database/sql`
- Pros: maximum control; most idiomatic Go; no third-party runtime dependency.
- Cons: significant boilerplate for scanning rows into structs; no migration tooling included; effectively reinvents what GORM or sqlc provide.

## Consequences

- **Positive:** schema changes during Phases 0-3 are fast -- update the struct, run auto-migrate, continue. Association loading (e.g., team with slots and moves) is concise.
- **Negative:** N+1 query risk on association loads -- must use `Preload` deliberately and verify generated SQL in the integration tests. Type safety is weaker than sqlc at the ORM boundary. If GORM's behavior becomes opaque or a query needs fine-grained control, raw SQL via `db.Raw()` is the escape hatch.

## Related Docs

- Tech spec section 3.2 (backend stack): `docs/tech-spec.md`
- Tech spec section 4 (alternatives -- sqlc noted as future revisit): `docs/tech-spec.md`
