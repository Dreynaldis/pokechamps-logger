# Wireframes: PokeChamps Logger

**Format:** ASCII mockups.
Text in `[BRACKETS]` = interactive element (button, tap target).
Text in `{braces}` = dynamic content filled at runtime.
`[ico]` = Pokemon icon (square sprite).

**Color convention (renders as actual border color in the real UI):**
- `╔══╗` double-line border = **blue** -- your Pokemon.
- `┌──┐` single-line border = **red** -- enemy Pokemon.
- In action lines: `[B: Name]` = blue (yours), `[R: Name]` = red (enemy).

---

## 1. Match History -- List

Each match is a card. Cards stack vertically.

```
┌─────────────────────────────────────────────────────────────┐
│  ╔═══════╗                                                  │
│  ║VICTORY║  <-- badge/sticker, top-left corner of card      │
│  ╚═══════╝                                                  │
│                                                             │
│  ╔═══╗╔═══╗╔═══╗       VS       ┌───┐┌───┐┌───┐           │
│  ╚═══╝╚═══╝╚═══╝                └───┘└───┘└───┘           │
│  ╔═══╗╔═══╗╔═══╗                ┌───┐┌───┐┌───┐           │
│  ╚═══╝╚═══╝╚═══╝                └───┘└───┘└───┘           │
│   (blue = yours)                  (red = enemy)            │
│  {Your team name}             {Jun 26, 2026}               │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  ╔═══════╗                                                  │
│  ║DEFEAT ║                                                  │
│  ╚═══════╝                                                  │
│                                                             │
│  ╔═══╗╔═══╗╔═══╗       VS       ┌───┐┌───┐┌───┐           │
│  ╚═══╝╚═══╝╚═══╝                └───┘└───┘└───┘           │
│  ╔═══╗╔═══╗╔═══╗                ┌───┐[ ? ][ ? ]           │
│  ╚═══╝╚═══╝╚═══╝                 (enemy not fully entered) │
│  {Your team name}             {Jun 26, 2026}               │
└─────────────────────────────────────────────────────────────┘
```

- Enemy slots not entered show a greyed `?` icon.
- Concede shows `DEFEAT` badge (same as loss).
- Tapping a card opens Match History Detail.

---

## 2. Team List (Home / My Teams)

Inspired by pokebase.app/pokemon-champions/teams -- a square card per team showing all 6 icons.

```
MY TEAMS                              [+ New Team]

┌──────────────────┐  ┌──────────────────┐
│  ★ ACTIVE        │  │                  │
│  {Team Alpha}    │  │  {Team Beta}     │
│                  │  │                  │
│ [ico][ico][ico]  │  │ [ico][ico][ico]  │
│ [ico][ico][ico]  │  │ [ico][ico][ico]  │
│                  │  │                  │
│ [Set Active] [Edit] │  │ [Set Active] [Edit] │
└──────────────────┘  └──────────────────┘
```

- Active team shows `★ ACTIVE` label and highlighted border.
- Only one team can be active at a time.

---

## 3. Team Builder

Inspired by pokebase.app/pokemon-champions/team-builder layout -- slot-based, one Pokemon per slot.

```
TEAM BUILDER -- {Team Alpha}          [Save Team]

┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐
│[ico] │  │[ico] │  │[ico] │  │[ico] │  │[ico] │  │  +   │
│      │  │      │  │      │  │      │  │      │  │      │
│Char. │  │Togek.│  │Landor│  │Incinr│  │Ursha │  │(empty│
└──────┘  └──────┘  └──────┘  └──────┘  └──────┘  └──────┘

  (tap a slot to expand it)

── Slot expanded: Charizard ──────────────────────────────────

  Species:  [Charizard               ▼]   [ico]
  Ability:  [Solar Power             ▼]
  Item:     [Charizardite Y          ▼]   → Mega Charizard Y
  Nature:   [Timid                   ▼]

  Moves (min 1, max 4):
  ┌──────────────────┐ ┌──────────────────┐
  │ Heat Wave        │ │ Air Slash        │
  └──────────────────┘ └──────────────────┘
  ┌──────────────────┐ ┌──────────────────┐
  │ + Add Move       │ │ + Add Move       │
  └──────────────────┘ └──────────────────┘

  Training Points: [ standard / custom ... ]

  [Remove Pokemon]
```

- Mega form derived from held item -- shown inline as a label, not a separate field.
- Slot shows `+` when empty; tap to search and add a Pokemon.
- Move slots filtered to species learnset (fallback: free text if pokebase data unavailable).

---

## 4. Pick Phase

Triggered when user starts a match against their active team.

```
PICK PHASE                              [Confirm Picks →]

YOUR TEAM (pick 4)         YOUR PICKS           ENEMY TEAM (optional)
╔═══════╗                ╔══════╦══════╗        ┌───────┐
║ [ico] ║ Charizard      ║ LEAD ║ LEAD ║        │ [ico] │ {fuzzy search}
║ [ico] ║ Togekiss   →   ║[ico] ║[ico] ║        │ [ico] │
║ [ico] ║ Landorus       ╠══════╬══════╣        │ [ico] │
║ [ico] ║ Incineroar     ║ BACK ║ BACK ║        │ [ico] │
║ [ico] ║ Urshifu        ║[ico] ║[ico] ║        │ [ ? ] │
║ [ico] ║ Rillaboom      ╚══════╩══════╝        │ [ ? ] │
╚═══════╝                (blue border)          └───────┘
                                                (red border)
                                                [+ Add Pokemon]
```

- Left column: your 6. Tap to add to picks (up to 4); tap again to remove.
- Center 2x2: top row = leads, bottom row = back. Drag or tap-to-assign from left column.
- Right column: enemy 6, entered via fuzzy search. Entirely optional.
- `[Confirm Picks →]` is enabled only when exactly 4 of your Pokemon are selected.

---

## 5. Turn Logger

No separate Turn 0.
Turn 1 starts immediately after Pick Phase.
Leads for both sides are set at the start of Turn 1 as part of entering the turn.

### 5a. Turn 1 -- entering leads + moves

Layout per side: `[bench small] → [active big] → [action buttons]`
Enemy is the mirror: `[action buttons] → [active big] → [bench small]`
Each row = one Pokemon. Two rows = two active slots per side.

```
TURN 1                                              [Log Turn →]  [End Match]

╔═══╗  ╔════════════╗  ┌──────────────┐  ║  ┌──────────────┐  ┌────────────┐  ┌───┐
║   ║  ║    [ico]   ║  │ [Move     ▼] │  ║  │ [Move     ▼] │  │  [Search]  │  │ ? │
║ico║  ║  {name}    ║  │ [Switch    ↕]│  ║  │ [Switch    ↕]│  │  {name}    │  │   │
╚═══╝  ╚════════════╝  └──────────────┘  ║  └──────────────┘  └────────────┘  └───┘

╔═══╗  ╔════════════╗  ┌──────────────┐  ║  ┌──────────────┐  ┌────────────┐  ┌───┐
║   ║  ║    [ico]   ║  │ [Move     ▼] │  ║  │ [Move     ▼] │  │  [Search]  │  │   │
║ico║  ║  {name}    ║  │ [Switch    ↕]│  ║  │ [Switch    ↕]│  │  {name}    │  │   │
╚═══╝  ╚════════════╝  └──────────────┘  ║  └──────────────┘  └────────────┘  └───┘

bench   active (blue)   our actions        enemy actions   active (red)   bench (red)
(blue)
```

- `║` center line = battle divider between our side and enemy side.
- Our bench (small, blue double-border) is auto-filled from pick phase (the 2 not chosen as leads).
- Enemy active slots are `[Search]` on Turn 1 -- user enters leads via fuzzy search. From Turn 2 onward they show the Pokemon name.
- Enemy bench `?` squares (small, red single-border) become named once a switch-in is logged.
- Each active slot (ours and enemy) has `[Move ▼]` and `[Switch ↕]`. Selecting one disables the other -- they are mutually exclusive per Pokemon per turn.
- When `[Switch ↕]` is chosen for a slot, the move dropdown greys out and the user picks which bench Pokemon switches in.

### 5b. Outcome entry (after moves are set, before logging)

After all moves are filled in, secondary prompts appear inline per action:

```
  [B: Charizard] → Heat Wave → [R: Landorus-T]
  ├─ Outcome: [Hit] [Miss] [Protected] [Immune]
  ├─ HP event: [+ Add HP event]      <-- repeatable
  │    └─ [ Which Pokemon? ▼ ] [ -__% HP ]
  └─ Secondary: [Burn? Y/N]          <-- shown only if move has secondary

  -- switch action (no move selected) --
  [B: Incineroar] switched out → [ Which Pokemon came in? ▼ ]

  -- pivot move (move selected AND triggers a switch) --
  [R: Incineroar] → Parting Shot → [B: Charizard]
  ├─ Outcome: [Hit] [Miss] [Protected] [Immune]
  ├─ Secondary: stat drop applied [Y/N]
  └─ Pivot switch-in: [ Which Pokemon came in? ▼ ]   <-- shown because Parting Shot is flagged as pivot
```

- HP events are discrete entries, one per source (primary damage, recoil, contact ability, etc.).
- Secondary effect prompt shown only when the move/ability/item is flagged for it in the DB.
- Pivot moves (Flip Turn, Volt Switch, U-turn, Parting Shot, etc.) are logged as a move, not a switch. The forced switch-in appears as a secondary prompt because those moves are flagged as pivot in the DB. This keeps move and switch mutually exclusive at the input level while still capturing the switch-in.
- Priority flag: small toggle `[!]` per action, off by default.

### 5c. Turn N (subsequent turns, same layout)

Same layout as 5a but enemy active slots now show names instead of `[Search]`, since leads were entered at Turn 1.
Enemy bench `?` slots become named once a switch-in is logged.
The logger carries forward known HP% from the previous turn's events.

---

## 6. Match History -- Detail

```
MATCH DETAIL -- {Jun 26 · Victory}                 [← Back]

YOUR TEAM:   ╔═╗╔═╗╔═╗╔═╗╔═╗╔═╗   (blue icons)
             ╚═╝╚═╝╚═╝╚═╝╚═╝╚═╝
ENEMY TEAM:  ┌─┐┌─┐┌─┐[ ? ][ ? ][ ? ]   (red icons)
             └─┘└─┘└─┘

────────────────────────────────────────────────────────────
TURN 1
  You brought:   [B: Charizard] · [B: Togekiss]  |  [B: Landorus] · [B: Incineroar] (bench)
  Enemy brought: [R: Landorus-T] · [R: Amoonguss]  |  [R: ?] · [R: ?] (bench)

  ACTION 1  [B: Charizard] → Heat Wave → [R: Landorus-T] [R: Amoonguss]
    Outcome: Hit
    HP: [R: Landorus-T] -45%,  [R: Amoonguss] -38%
    Secondary: Burn ([R: Landorus-T])

  ACTION 2  [B: Togekiss] → Air Slash → [R: Amoonguss]
    Outcome: Hit
    HP: [R: Amoonguss] -30%
    Secondary: Flinch

  ACTION 3  [R: Landorus-T] → Rock Slide  [!] priority
    Outcome: Protected ([B: Charizard] blocked)

  ACTION 4  [R: Amoonguss] → Spore → [B: Charizard]
    Outcome: Immune

────────────────────────────────────────────────────────────
TURN 2
  ...

────────────────────────────────────────────────────────────
RESULT: VICTORY
```

- Actions listed in priority order as logged.
- HP events shown in the order they occurred within that action.
- `[!]` badge marks priority moves.
- Text-based in v1; HP% events are the data layer for post-MVP visual bars.

---

## Notes & Open Decisions

| Decision | Notes |
|---|---|
| Frontend framework | Not yet chosen -- affects component patterns for autocomplete, drag-to-assign in pick phase |
| Pokemon icon source | Need sprite URLs (PokeAPI has official sprites) |
| Drag vs tap-to-assign in pick phase | Drag is more natural on desktop; tap-sequence (tap your Pokemon, tap target slot) may be better on mobile |
| Enemy move logging | Enemy has 2 move dropdowns per active -- log both or just one? Current design logs what they actually used, one move per turn per Pokemon |
| Turn scroll vs paginated | Detail view: scroll is simpler; paginated (prev/next turn) may be faster to navigate long matches |
