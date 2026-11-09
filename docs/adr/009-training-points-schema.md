# ADR 009 -- Training Points Schema + bits-ui Adoption

## Status

Accepted

## Context

`team_slots.training_points` has existed as an untyped `jsonb` column since Phase 3 Layer 1, with the schema explicitly deferred ("EV-equivalent; exact shape TBD once Champions UI is confirmed" -- see ADR 001, tech-spec.md, PROGRESS.md open questions).

While testing the team builder, the actual Pokemon Champions mechanic was confirmed directly by the user:

- Each Pokemon has a pool of **66 total training points**.
- Points can be spread freely across the 6 stats (HP, Attack, Defense, Sp. Atk, Sp. Def, Speed) -- there is no requirement to max any particular stat.
- No single stat can receive more than **32** training points.
- A point adds directly to the base stat -- there is no divisor or level-dependent scaling (unlike mainline Pokemon's EV system, where 4 EVs = +1 stat at level 100).

Separately, the same round of testing surfaced the need for fuzzy-searchable pickers (items, moves) and stat-allocation sliders, none of which the codebase had a pattern for. `docs/design-system.md` already named **bits-ui** as the preferred headless component library for comboboxes and select replacements, but it had never actually been adopted -- the existing Pokemon-species search was a fully hand-built dropdown.

## Decision

**Training points schema:**

```go
type TrainingPointsJSON struct {
	HP      int `json:"hp"`
	Attack  int `json:"attack"`
	Defense int `json:"defense"`
	SpAtk   int `json:"sp_atk"`
	SpDef   int `json:"sp_def"`
	Speed   int `json:"speed"`
}
```

Marshaled into the existing `team_slots.training_points jsonb` column -- no migration needed, the column already exists. Validation lives in `internal/handler/team.go`'s `validateSlots`: each stat must be in `0-32`, and the sum across all six must be `<= 66`. Effective stat is computed client-side and (later, for match logging) server-side as `base_stat + training_points`, no divisor.

**bits-ui adoption:** adopted now rather than continuing hand-built dropdowns, since the team builder redesign needs three new fuzzy-searchable pickers (Pokemon species, items, moves) plus replacing the ability/nature `<select>` elements. Building three more one-off custom dropdowns would have compounded the inconsistency the design doc already flagged. bits-ui is headless (no imposed visual style), so the existing dark-theme design tokens continue to apply unchanged. This is the first component-library dependency in the frontend; it targets Svelte 5, matching the existing `svelte ^5.0.0` requirement.

**Exception: `StatSlider.svelte` does not use bits-ui's `Slider`.** The training-points UI has a requirement bits-ui's Slider architecture can't express: the track must always render at full length representing the fixed 0-32 scale, but dragging must hard-stop at a much smaller, per-stat remaining budget (e.g. 2 of 32, once other stats have used up the shared 66-point pool). bits-ui ties both the drag-to-value math and the visual thumb/fill position to a single `max` prop -- every combination tried (budget-scaled `max`, fixed `max` with a CSS position override, etc.) broke either the interaction or the visual, because dragging and rendering share one coordinate system in that implementation. `StatSlider.svelte` instead uses a native `<input type="range" min=0 max=32>`, clamping the budget by writing the DOM's own `.value` back synchronously inside the same `input` event (no framework round-trip) -- this keeps drag distance and visual position always 1:1 against the true 32-point scale while making an overshoot physically impossible mid-drag.

## Consequences

- `training_points` is no longer a fully open JSONB shape -- it's app-validated to the six known stat keys, consistent with how `base_stats` and `types` are already treated (documented shape, enforced at the application layer per ADR 001).
- Any future stat-related feature (e.g. showing effective stats in the match logger) can rely on this fixed shape without re-deriving it.
- The existing hand-built Pokemon-search dropdown, and the plain `<select>` elements for ability/nature/item, are migrated onto shared bits-ui-based wrapper components (`Combobox.svelte`, `Select.svelte`) instead of remaining a mix of patterns.
- A `frontend/src/lib/components/` directory now exists as the convention for shared UI components going forward.
- bits-ui adoption is not absolute -- `StatSlider.svelte` is a documented, deliberate exception. Future components with a similar "visual scale independent of interactive range" requirement should default to a native control rather than assuming bits-ui covers every case.
