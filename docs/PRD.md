# PRD: PokeChamps Logger

**Status:** Draft · **Last updated:** 2026-06-26

## 1. Overview / Vision

PokeChamps Logger is a web-based battle documentation tool for Pokemon Champions (Switch/mobile) online players.
It lets players record their team, the pick phase, and every turn of a match -- filling the gap left by the game having no replay feature.
The goal is to give players a structured, fast-to-input record they can review after the match to understand what happened, adapt to the meta, and improve their team tuning over time.

## 2. Success Criteria & Metrics

| Metric | Target | Why |
|---|---|---|
| Turn logging input speed | Under 20 seconds per turn | Must not fall behind a live match |
| Retention | 70% of users who create a team complete at least 3 logged matches | 3 matches signals the tool is becoming a habit, not a one-off |
| Team builder completion | User reaches battle phase after building a team | Core funnel -- if this breaks, nothing else works |
| v1 launch gate | Team builder + battle logger + match history functional end-to-end | All three must ship together |

## 3. Users / Personas

**Persona 1 -- The Competitive Grinder**
Plays 7-10 ranked matches per week on Switch or mobile.
Actively tries to climb the ranked ladder.
Wants to understand why they lost: which Pokemon the enemy brought, how the speed tiers played out, whether their held item choices were correct.
Currently has no way to review matches -- relies on memory alone.

**Persona 2 -- The Improving Beginner**
New to competitive doubles, unfamiliar with terminology like "Trick Room," "priority," or "speed tuning."
Plays casually but wants to get better.
Needs the UI to be readable without jargon, with inline explanation where needed.
Would benefit most from reviewing turn order and outcomes to understand what went wrong.

**Not targeted (v1):** Tournament players (secondary devices banned at events).

## 4. Problem Statement

Pokemon Champions has no in-game replay system.
Players who want to study their matches -- typically 1-3 per day -- have no structured way to record what happened: which Pokemon were brought, what moves were used, in what order, and what the outcomes were.
Without a record, post-match review is limited to memory, which degrades quickly and misses the detail needed to make real adjustments.

The practical loss is significant: players cannot review damage distribution, held item effectiveness, speed tuning decisions, or how to adapt to the current meta by seeing how others build their Pokemon.
The only comparable tool is Wolfeyvgc's paper difference sheet for VGC -- a physical form, not designed for Pokemon Champions, and not usable on a device.
There is no digital equivalent.

## 5. Core Features (problem → solution)

| Feature | Problem it solves | Acceptance criteria |
|---|---|---|
| **Team Builder** | Players need a reusable record of their queued team so battle logging can reference it without re-entry each match. | 6 Pokemon registered; each has at least 1 move selected. Held item and 4 moves are not required -- some Pokemon run no item or fewer moves. Mega form is not a separate field -- it is derived from the held mega stone (e.g. Charizard + Charizardite Y = Mega Charizard Y). One team marked active at a time. |
| **Pick Phase Logger** | Enemy's 6 Pokemon are visible during pick phase but lost after -- no in-game record. | User selects their 4 from active team's 6. Enemy's 6 entry is optional (time-pressured). Fuzzy search autocomplete over 310 Pokemon by name, results update within 100ms per keystroke. |
| **Turn 0 -- Leads** | Which Pokemon led and which stayed back shapes the entire match; this is lost without a record. | User picks their 2 leads from the 4 they brought. Enemy leads entered from pre-entered list or fresh fuzzy search. Enemy back 2 marked unknown until switched in. |
| **Turn Logger (Turn 1+)** | No in-game record of moves used, order, or outcomes. | Up to 4 actions per turn logged (2 per side, priority-ordered). Each action captures: Pokemon, move, priority flag (yes/no), outcome (hit / miss / protected / immune). If the move type matches a known ability immunity on the target, the logger prompts "no effect?" proactively. Secondary effects prompted only when move has them: stat changes, status conditions, weather, field effects, held item duration interactions (e.g. Light Clay, Damp Rock), mega evolution (who, which form). Every HP-related event within an action is logged as a separate discrete entry -- primary damage, recoil, contact ability damage, and any other source each get their own prompt (which Pokemon, how much HP% lost). This keeps the health bar data accurate and renders each HP change as its own step in the post-MVP visualization. |
| **Match End** | Matches end by KO or concede; result must be attached to the log. | End options: win / loss / concede. Concede stored as loss in history list view. |
| **Match History -- List** | Players need to locate a past match quickly. | Each row shows: your team (6 Pokemon), enemy team (up to 6, whatever was recorded), result (win/loss). |
| **Match History -- Detail** | Players need to step through a match turn by turn to understand decisions. | Turn-based list. Turn 0 shows leads. Each subsequent turn lists up to 4 actions, one per Pokemon. Each action line shows: Pokemon name, move used, outcome (hit/miss/protected/immune), HP% events in the order they occurred (each source listed separately -- primary damage, recoil, contact ability damage, etc.), and any secondary effects logged (stat change, status condition, weather change, field effect, mega evolution). |

## 6. User Flow

**New user:**
```
Sign up
      |
Create a team (Team Builder)
  -- 6 Pokemon, each with at least 1 move
  -- Set team as active
      |
[continues to Start a match below]
```

**Returning user:**
```
Log in
      |
Set active team (or keep existing active team)
      |
[continues to Start a match below]
```

**Start a match (shared path):**
```
Start a match
  -- Select 4 Pokemon to bring from active team's 6
  -- (Optional) Enter enemy's 6 Pokemon via fuzzy search
      |
Turn 0 -- Leads
  -- Pick your 2 leads from your 4
  -- Enter enemy's 2 leads
      |
Turn 1, 2, 3 ... (repeat until match ends)
  -- Log up to 4 actions per turn
  -- Secondary effect prompts appear only when relevant
      |
End match
  -- Mark win / loss / concede
      |
       +--> Review match in Match History
       +--> Start next match
```

## 7. Data & Privacy

- Match data is personal and private -- each user can only see their own match history.
- No match data is shared publicly or with other users in v1.
- No automatic expiry in v1 -- retention policy to be decided once usage patterns and storage costs are known.

## 8. Performance & Platform Constraints

- **Platform:** Web-based, PWA-compatible. Must be fully usable on mobile (the primary logging device during a match).
- **Pokemon/move search:** Implemented as an autocomplete component. List renders alphabetically by default with a search field at the top. Results must update within 100ms of each keystroke. This is the most latency-sensitive interaction in the app and must be validated on mid-range mobile hardware.
- **Turn input speed target:** Full turn logged in under 20 seconds on mobile.
- **Offline:** Not required for v1, but PWA shell should load fast on a mobile network.

## 9. Non-Goals (v1)

- Tournament mode or offline/print mode.
- Damage calculator (predicting damage output from stats, EVs, and move data). HP% changes are recorded as user-reported observations, not computed.
- Opponent identity tracking (no username or ranking capture).
- Move suggestions, AI analysis, or coaching features.
- Visual match analytics and team stats (post-MVP -- see Section 12).
- Learnset validation in team builder (blocked on data access -- see Risks).

## 10. Open Questions

| Question | Owner | Status |
|---|---|---|
| Can we scrape or access pokebase.app data (learnsets, movesets, abilities per Pokemon)? | Dev | Open -- outreach to pokebase required before implementation. Ask for scrape permission or API access; non-commercial use; offer credit as data source. Fallback: free-text move entry (no filtering). |
| Tech stack -- backend, frontend, database | Dev | Open -- all three to be decided in tech spec phase. |
| Retention policy | Dev/Product | Open -- no auto-expiry in v1. Revisit once storage costs and usage patterns are known. |

## 11. Risks & Blockers

**Data access (blocker):**
The team builder's move slot filtering and the 310-Pokemon list depend on Champions-specific learnset data from pokebase.app.
If permission is denied and no alternative source is found, move filtering cannot be built as designed.
Fallback: free-text move entry degrades UX but unblocks launch.

**Data maintenance:**
Pokemon Champions patches roughly monthly.
New Pokemon or move changes must be applied to the Champions override layer within a patch cycle or the team builder shows stale data.
This is an ongoing operational responsibility.

**Time pressure UX:**
Turn logging is real-time by design but can fall back to post-match reconstruction.
If input proves too slow in testing with real players, the secondary effect prompts are the first candidate to simplify.
Must be validated with real players before hardening the turn logger UI.

**Data layer:**
Move database seeded from PokeAPI (Gen 1-9 moves: type, base power, secondary effect flag).
Any move, ability, or held item that has underlying battle effects is flagged as such in the database. This unified secondary effects flag is what drives the turn logger's contextual prompts -- e.g. "no effect?" when an ability grants immunity, additional HP events when an ability or move causes passive damage, duration extensions when a held item modifies a field effect. The tech spec will define how these are structured per entity type; at the PRD level they are treated as a single concept.
Champions-specific override layer handles learnset differences and mega stone-to-form mappings (e.g. Charizardite X → Mega Charizard X).
Patch maintenance falls on the dev/maintainer.

## 12. Post-Launch / Iteration Log

**Planned post-MVP:**

- **Visual match review:** In the match detail view, add visual bars (health bar style, damage bar style) to represent damage dealt and received per turn -- more game-native than raw numbers or charts.
  This is a presentation layer on top of the existing turn log data, not a separate analytics pipeline.

- **Team stats (unlocks at 100 logged matches):** Once a user has 100 completed matches on record, their registered teams surface aggregate stats: win rate per team, most-used lead pairs, most-encountered enemy Pokemon.
  The 100-match threshold ensures the stats are meaningful before being shown.

- **Speed tuning and stat distribution review tools.**
- **Export or share match logs.**

**Changelog:**
| Date | Change |
|---|---|
| 2026-06-26 | Initial draft from design session. |
| 2026-06-26 | Added success metrics, personas, user flow, data/privacy section, performance constraints, acceptance criteria per feature. |
| 2026-06-26 | Added immune as 4th turn outcome; ability immunity data requirement; returning user flow; 100ms search spec; stale retention/deletion lines removed. |

## 13. Related Docs

- Tech Spec: _not yet written_
- ADRs: `docs/adr/` _(none yet -- to be written when tech stack and data access are finalized)_
- Reference: [Wolfeyvgc Difference Sheet](https://www.figma.com/community/file/1622016964100175367/wolfeyvgcs-difference-sheet) (design inspiration)
- Reference: [pokebase.app Pokemon Champions](https://pokebase.app/pokemon-champions/pokemon) (data source candidate)
