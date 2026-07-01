# ADR 008 -- Persistence Strategy (Auto-save, Session Recovery, Mid-match Detection)

## Status

Accepted

## Context

Two flows in this app require persistent state that must survive accidental tab closes, refreshes, and navigation away:

**Team builder** -- user fills 6 Pokemon slots with species, moves, items, abilities.
This can take several minutes and has no natural "done" signal until the user explicitly activates the team.
A misclick closing the tab should not lose this work.

**Match logger** -- user logs turns in real time during an active match.
Losing a partially-logged match is the worst failure mode in the app.
Each turn is a discrete action (not free-form text), so persistence needs to be nearly immediate.

The requirement is: **no Save button anywhere in the app**.
Changes are persisted automatically.
On return to the app (same session or new tab), in-progress work is resumed without prompting the user to re-enter anything.

## Decision

Use a **write-through hybrid**: localStorage is the immediate write target; the server is the durable record.

```
User action
    │
    ▼
localStorage (immediate, synchronous write)
    │
    ▼ debounced background sync
Server (durable, cross-device)
```

On app load, the server is checked first.
If the server has a newer timestamp than localStorage, server data wins.
If localStorage is newer (server sync lagged), localStorage data is used and re-synced.

This means:
- Writes feel instant -- no spinner, no waiting for network
- Data survives browser storage clears -- server is the backup
- Works if the user is briefly offline -- localStorage holds the buffer
- Cross-device recovery works -- server is the canonical record

## Persistence Rules Per Flow

### Team Builder

- **Trigger:** any field change (species, move slot, item, ability)
- **localStorage write:** debounced 300ms after last change. Key: `team-draft:{teamId}` for existing teams, `team-draft:new` for unsaved drafts.
- **Server sync:** debounced 1500ms after last change. `PATCH /api/v1/teams/:id` with full team payload. New drafts are created on first sync via `POST /api/v1/teams`.
- **On page load:** fetch team from server. If localStorage has a newer `updatedAt` for the same team, use localStorage and queue a sync. If equal or server is newer, use server data and clear the localStorage entry.
- **Draft teams (not yet named):** stored under `team-draft:new`. Promoted to a real team on first server sync. Until then they only exist in localStorage.

### Match Logger

Match turns are discrete events, not free-form edits.
Each turn submission is sent to the server immediately -- no debounce.

- **Pick phase:** `POST /api/v1/matches` creates the match record (`status: draft`). Pick selections are written to localStorage (`match-active`) and synced immediately on each pick action.
- **Turn submission:** each turn action (`POST /api/v1/matches/:id/turns`) fires immediately on user confirm. localStorage is updated in parallel. No debounce -- losing a turn is not acceptable.
- **Match status transitions:** `draft → in_progress` on first turn submission. `in_progress → completed` on match-end. These are server-authoritative -- localStorage reflects but does not drive status.
- **localStorage key:** `match-active` (only one active match at a time). Stores `{ matchId, status, currentTurn, localTurns[] }`.

## Mid-match Detection

On every app load, regardless of which page the user lands on, the following check runs once:

1. Read `match-active` from localStorage.
2. If it exists and `status === 'in_progress'`, call `GET /api/v1/matches/:id`.
3. Server confirms `in_progress` -- show a persistent banner at the top of the page: "You have an unfinished match. [Resume]". Banner stays until dismissed or match is completed.
4. Server says `completed` -- clear `match-active` from localStorage. No banner.
5. Server returns 404 -- clear localStorage entry. No banner.
6. Server unreachable -- show the banner anyway using localStorage data. User can attempt to resume; retry sync when connection restores.

The banner is non-blocking -- the user can continue navigating freely.
Clicking "Resume" navigates to the match logger at the current turn.

## Conflict Resolution

**Two tabs open on the same team:**
Last write wins in localStorage (tabs share the same storage).
Server sync from both tabs will race; last `PATCH` to arrive wins.
This is acceptable -- two-tab editing of the same team is an unsupported edge case.

**Two devices (cross-device):**
Device B opens the app with stale localStorage.
Device B fetches the team from the server (fresher `updatedAt`), overwrites its own localStorage, and renders server data.
Server is always the tiebreaker across devices.

## What Is Not Stored in localStorage

- Auth tokens -- access token is in memory; refresh token is an HttpOnly cookie.
- Pokemon roster data -- read-only reference data, always fetched from the server.
- Completed match history -- completed matches live on the server only; no client-side cache in v1.

## UI Conventions

- No Save button anywhere in the app.
- A subtle auto-save indicator appears after each successful server sync -- a brief "Saved" toast (bottom-right, fades out after 1.5s via svelte-sonner). No user action required.
- If server sync fails, the toast shows "Saved locally -- syncing when online" and retries silently with exponential backoff (max 30s interval).
- Destructive actions (delete team, abandon match) use a bits-ui confirmation modal -- never `window.confirm`.

## Tradeoffs

| Concern | Impact |
|---|---|
| localStorage is per-device | Cross-device recovery requires at least one successful server sync before the session was lost. Acceptable -- server is always the durable record. |
| 1500ms server sync lag on team edits | A crash in that window loses up to 1.5s of team edits. Acceptable for team building; match turns are unaffected (no debounce). |
| Two-tab conflicts | Last-write-wins is lossy in theory. Not a real scenario -- one active match and one active team editor at a time by design. |
| localStorage quota | Team draft ~5KB, active match state ~20KB for a 30-turn match. Well within the 5MB typical quota. |
