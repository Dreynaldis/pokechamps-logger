<script lang="ts">
	import { onMount } from 'svelte';
	import { authState } from '$lib/auth.svelte.js';
	import { api } from '$lib/api.js';
	import type { Team, DraftTeam, DraftSlot, DraftMove, PokemonSummary, PokemonDetail } from '$lib/types.js';

	// ---------------------------------------------------------------------------
	// State
	// ---------------------------------------------------------------------------

	let teams = $state<Team[]>([]);
	let selectedTeamID = $state<string | null>(null);
	let draft = $state<DraftTeam | null>(null);
	let saveStatus = $state<'saved' | 'saving' | 'unsaved'>('saved');
	let teamsLoading = $state(true);

	type SlotSearch = { query: string; results: PokemonSummary[]; open: boolean };
	let slotSearch = $state<SlotSearch[]>(Array.from({ length: 6 }, () => ({ query: '', results: [], open: false })));
	let slotDetail = $state<Array<PokemonDetail | null>>(Array(6).fill(null));

	let lsTimer: ReturnType<typeof setTimeout>;
	let serverTimer: ReturnType<typeof setTimeout>;
	let searchTimers: Array<ReturnType<typeof setTimeout>> = Array(6).fill(0);

	const NATURES = ['Hardy','Lonely','Brave','Adamant','Naughty','Bold','Docile','Relaxed',
		'Impish','Lax','Timid','Hasty','Serious','Jolly','Naive','Modest','Mild',
		'Quiet','Bashful','Rash','Calm','Gentle','Sassy','Careful','Quirky'];

	// ---------------------------------------------------------------------------
	// Boot
	// ---------------------------------------------------------------------------

	onMount(async () => {
		if (!authState.user) { window.location.href = '/auth'; return; }
		await loadTeams();
	});

	async function loadTeams() {
		teamsLoading = true;
		const res = await api('/api/v1/teams');
		if (res.ok) teams = await res.json();
		teamsLoading = false;
	}

	// ---------------------------------------------------------------------------
	// Auto-save (explicit, not $effect -- avoids false dirty on team load)
	// ---------------------------------------------------------------------------

	function markDirty() {
		saveStatus = 'unsaved';
		clearTimeout(lsTimer);
		clearTimeout(serverTimer);
		lsTimer = setTimeout(() => {
			if (draft) localStorage.setItem(`team-draft:${draft.id}`, JSON.stringify(draft));
		}, 300);
		serverTimer = setTimeout(async () => {
			saveStatus = 'saving';
			await syncToServer();
		}, 1500);
	}

	async function syncToServer() {
		if (!draft) return;
		const body = {
			name: draft.name,
			slots: draft.slots
				.filter(s => s.pokemon !== null)
				.map(s => ({
					slot: s.slot,
					pokemon_id: s.pokemon!.id,
					ability_id: s.abilityID ?? undefined,
					nature: s.nature ?? undefined,
					moves: s.moves
						.map((m, i) => m ? { slot: i + 1, move_id: m.moveID } : null)
						.filter(Boolean),
				})),
		};
		const res = await api(`/api/v1/teams/${draft.id}`, {
			method: 'PATCH',
			body: JSON.stringify(body),
		});
		if (res.ok) {
			saveStatus = 'saved';
			const listRes = await api('/api/v1/teams');
			if (listRes.ok) teams = await listRes.json();
		}
	}

	// ---------------------------------------------------------------------------
	// Team selection + load
	// ---------------------------------------------------------------------------

	async function selectTeam(id: string) {
		selectedTeamID = id;
		const res = await api(`/api/v1/teams/${id}`);
		if (!res.ok) return;
		await buildDraft(await res.json());
	}

	async function buildDraft(team: Team) {
		const emptySlots: DraftSlot[] = Array.from({ length: 6 }, (_, i) => ({
			slot: i + 1, pokemon: null, abilityID: null, nature: null,
			moves: [null, null, null, null],
		}));

		const newSearch: SlotSearch[] = Array.from({ length: 6 }, () => ({ query: '', results: [], open: false }));
		const newDetail: Array<PokemonDetail | null> = Array(6).fill(null);

		for (const s of team.slots ?? []) {
			const idx = s.slot - 1;
			const detailRes = await api(`/api/v1/pokemon/${s.pokemon.name}`);
			const detail: PokemonDetail = detailRes.ok ? await detailRes.json() : s.pokemon as unknown as PokemonDetail;
			newDetail[idx] = detail;
			newSearch[idx].query = s.pokemon.display_name;

			const moves: Array<DraftMove | null> = [null, null, null, null];
			for (const m of s.moves) {
				moves[m.slot - 1] = { slot: m.slot, moveID: m.move.id, displayName: m.move.display_name };
			}
			emptySlots[idx] = {
				slot: s.slot, pokemon: detail,
				abilityID: s.ability?.id ?? null, nature: s.nature ?? null, moves,
			};
		}

		slotSearch = newSearch;
		slotDetail = newDetail;
		draft = { id: team.id, name: team.name, slots: emptySlots };
		saveStatus = 'saved';
	}

	// ---------------------------------------------------------------------------
	// Team CRUD
	// ---------------------------------------------------------------------------

	async function createTeam() {
		const name = `Team ${teams.length + 1}`;
		const res = await api('/api/v1/teams', { method: 'POST', body: JSON.stringify({ name }) });
		if (!res.ok) return;
		const t: Team = await res.json();
		teams = [...teams, t];
		await selectTeam(t.id);
	}

	async function deleteTeam(id: string) {
		if (!confirm('Delete this team?')) return;
		await api(`/api/v1/teams/${id}`, { method: 'DELETE' });
		teams = teams.filter(t => t.id !== id);
		if (selectedTeamID === id) { selectedTeamID = null; draft = null; }
	}

	async function activateTeam(id: string) {
		await api(`/api/v1/teams/${id}/activate`, { method: 'POST' });
		teams = teams.map(t => ({ ...t, is_active: t.id === id }));
	}

	// ---------------------------------------------------------------------------
	// Slot editing -- each mutator calls markDirty() explicitly
	// ---------------------------------------------------------------------------

	async function onPokemonSearch(idx: number) {
		const q = slotSearch[idx].query;
		clearTimeout(searchTimers[idx]);
		if (!q) { slotSearch[idx].results = []; slotSearch[idx].open = false; return; }
		searchTimers[idx] = setTimeout(async () => {
			const res = await api(`/api/v1/pokemon?q=${encodeURIComponent(q)}&limit=8`);
			if (res.ok) {
				slotSearch[idx].results = await res.json();
				slotSearch[idx].open = slotSearch[idx].results.length > 0;
			}
		}, 200);
	}

	async function selectPokemon(idx: number, p: PokemonSummary) {
		slotSearch[idx].query = p.display_name;
		slotSearch[idx].open = false;
		const detailRes = await api(`/api/v1/pokemon/${p.name}`);
		if (!detailRes.ok) return;
		const detail: PokemonDetail = await detailRes.json();
		slotDetail[idx] = detail;
		draft!.slots[idx] = {
			slot: idx + 1, pokemon: detail,
			abilityID: detail.abilities[0]?.id ?? null, nature: null,
			moves: [null, null, null, null],
		};
		draft = { ...draft! };
		markDirty();
	}

	function clearSlot(idx: number) {
		slotSearch[idx] = { query: '', results: [], open: false };
		slotDetail[idx] = null;
		draft!.slots[idx] = { slot: idx + 1, pokemon: null, abilityID: null, nature: null, moves: [null, null, null, null] };
		draft = { ...draft! };
		markDirty();
	}

	function setAbility(idx: number, abilityID: string) {
		draft!.slots[idx] = { ...draft!.slots[idx], abilityID: abilityID || null };
		draft = { ...draft! };
		markDirty();
	}

	function setNature(idx: number, nature: string) {
		draft!.slots[idx] = { ...draft!.slots[idx], nature: nature || null };
		draft = { ...draft! };
		markDirty();
	}

	function setMove(idx: number, moveSlot: number, moveID: string) {
		const move = slotDetail[idx]?.moves.find(m => m.id === moveID);
		const moves = [...draft!.slots[idx].moves];
		moves[moveSlot] = moveID ? { slot: moveSlot + 1, moveID, displayName: move?.display_name ?? '' } : null;
		draft!.slots[idx] = { ...draft!.slots[idx], moves };
		draft = { ...draft! };
		markDirty();
	}

	function onTeamNameInput() {
		markDirty();
	}

	const typeColors: Record<string, string> = {
		normal:'#A8A878', fire:'#F08030', water:'#6890F0', electric:'#F8D030',
		grass:'#78C850', ice:'#98D8D8', fighting:'#C03028', poison:'#A040A0',
		ground:'#E0C068', flying:'#A890F0', psychic:'#F85888', bug:'#A8B820',
		rock:'#B8A038', ghost:'#705898', dragon:'#7038F8', dark:'#705848',
		steel:'#B8B8D0', fairy:'#EE99AC',
	};
</script>

<div class="page">
	<!-- Left: team list -->
	<aside class="team-list">
		<div class="team-list-header">
			<span class="section-label">Teams</span>
			<button class="btn-icon" onclick={createTeam} title="New team">+</button>
		</div>

		{#if teamsLoading}
			<p class="muted p-sm">Loading...</p>
		{:else if teams.length === 0}
			<p class="muted p-sm">No teams yet.</p>
		{:else}
			{#each teams as t}
				<button
					class="team-item"
					class:selected={t.id === selectedTeamID}
					onclick={() => selectTeam(t.id)}
				>
					<span class="team-name-text">{t.name}</span>
					{#if t.is_active}<span class="active-badge">Active</span>{/if}
				</button>
			{/each}
		{/if}
	</aside>

	<!-- Right: team builder -->
	<div class="builder">
		{#if !selectedTeamID}
			<div class="empty-state">
				<p>Select a team or <button class="link-btn" onclick={createTeam}>create one</button></p>
			</div>
		{:else if draft}
			<!-- Team header -->
			<div class="team-header">
				<input
					class="team-name-input"
					bind:value={draft.name}
					oninput={onTeamNameInput}
					placeholder="Team name"
				/>
				<div class="team-actions">
					<span
						class="save-status"
						class:saving={saveStatus === 'saving'}
						class:unsaved={saveStatus === 'unsaved'}
					>
						{saveStatus === 'saved' ? '✓ Saved' : saveStatus === 'saving' ? 'Saving...' : '● Unsaved'}
					</span>
					<button
						class="btn-sm"
						onclick={() => activateTeam(draft!.id)}
						disabled={teams.find(t => t.id === draft!.id)?.is_active}
					>
						{teams.find(t => t.id === draft!.id)?.is_active ? 'Active' : 'Set active'}
					</button>
					<button class="btn-sm danger" onclick={() => deleteTeam(draft!.id)}>Delete</button>
				</div>
			</div>

			<!-- 6 Slots -->
			<div class="slots-grid">
				{#each draft.slots as slot, idx}
					{@const detail = slotDetail[idx]}
					<div class="slot-card">
						<div class="slot-number">Slot {slot.slot}</div>

						<!-- Pokemon search -->
						<div class="field-group">
							<span class="field-label">Pokemon</span>
							<div class="autocomplete">
								<div class="autocomplete-input-row">
									<input
										class="field-input"
										bind:value={slotSearch[idx].query}
										oninput={() => onPokemonSearch(idx)}
										onfocus={() => { if (slotSearch[idx].results.length) slotSearch[idx].open = true; }}
										onblur={() => setTimeout(() => { slotSearch[idx].open = false; }, 150)}
										placeholder="Search Pokemon..."
										autocomplete="off"
									/>
									{#if slot.pokemon}
										<button class="clear-btn" onclick={() => clearSlot(idx)}>✕</button>
									{/if}
								</div>

								{#if slotSearch[idx].open && slotSearch[idx].results.length}
									<ul class="dropdown">
										{#each slotSearch[idx].results as p}
											<li>
												<button onmousedown={() => selectPokemon(idx, p)}>
													<span>{p.display_name}</span>
													<span class="types-inline">
														{#each p.types as t}
															<span class="type-pip" style="background:{typeColors[t] ?? '#888'}">{t}</span>
														{/each}
													</span>
												</button>
											</li>
										{/each}
									</ul>
								{/if}
							</div>

							{#if slot.pokemon}
								<div class="types-row">
									{#each slot.pokemon.types as t}
										<span class="type-badge" style="background:{typeColors[t] ?? '#888'}">{t}</span>
									{/each}
								</div>
							{/if}
						</div>

						{#if slot.pokemon && detail}
							<!-- Ability -->
							<div class="field-group">
								<span class="field-label">Ability</span>
								<select
									class="field-select"
									value={slot.abilityID ?? ''}
									onchange={e => setAbility(idx, (e.target as HTMLSelectElement).value)}
								>
									<option value="">-- none --</option>
									{#each detail.abilities as a}
										<option value={a.id}>{a.display_name}</option>
									{/each}
								</select>
							</div>

							<!-- Nature -->
							<div class="field-group">
								<span class="field-label">Nature</span>
								<select
									class="field-select"
									value={slot.nature ?? ''}
									onchange={e => setNature(idx, (e.target as HTMLSelectElement).value)}
								>
									<option value="">-- none --</option>
									{#each NATURES as n}
										<option value={n}>{n}</option>
									{/each}
								</select>
							</div>

							<!-- Moves -->
							<div class="field-group">
								<span class="field-label">Moves</span>
								<div class="moves-list">
									{#each [0, 1, 2, 3] as mi}
										<select
											class="field-select move-select"
											value={slot.moves[mi]?.moveID ?? ''}
											onchange={e => setMove(idx, mi, (e.target as HTMLSelectElement).value)}
										>
											<option value="">Move {mi + 1}</option>
											{#each detail.moves as m}
												<option value={m.id}>{m.display_name} ({m.type})</option>
											{/each}
										</select>
									{/each}
								</div>
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.page { display: grid; grid-template-columns: 200px 1fr; height: calc(100vh - 52px); }

	/* Team list */
	.team-list { border-right: 1px solid #1e1e35; overflow-y: auto; display: flex; flex-direction: column; }
	.team-list-header {
		display: flex; align-items: center; justify-content: space-between;
		padding: 0.75rem 0.875rem; border-bottom: 1px solid #1e1e35;
	}
	.section-label { font-size: 0.75rem; color: #666; text-transform: uppercase; letter-spacing: 0.06em; }
	.btn-icon {
		background: transparent; border: 1px solid #2a2a45; border-radius: 5px;
		color: #888; cursor: pointer; font-size: 1rem; width: 26px; height: 26px;
		display: flex; align-items: center; justify-content: center; line-height: 1;
	}
	.btn-icon:hover { color: #e0e0e0; border-color: #555; }

	.team-item {
		width: 100%; text-align: left; padding: 0.6rem 0.875rem;
		background: transparent; border: none; color: #aaa; cursor: pointer;
		font-size: 0.85rem; display: flex; align-items: center; gap: 0.5rem;
	}
	.team-item:hover { background: #1a1a2e; color: #e0e0e0; }
	.team-item.selected { background: #1e1e35; color: #e0e0e0; }
	.team-name-text { flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.active-badge {
		font-size: 0.65rem; padding: 1px 5px; border-radius: 3px;
		background: #1e3a1e; color: #6bcc8a; white-space: nowrap;
	}

	/* Builder area */
	.builder { overflow-y: auto; padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem; }

	.empty-state {
		display: flex; align-items: center; justify-content: center;
		height: 100%; color: #666; font-size: 0.9rem;
	}
	.link-btn { background: none; border: none; color: #a5b4fc; cursor: pointer; font-size: inherit; padding: 0; }
	.link-btn:hover { text-decoration: underline; }

	/* Team header */
	.team-header { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; }
	.team-name-input {
		flex: 1; min-width: 160px; padding: 0.4rem 0.6rem; border-radius: 6px;
		border: 1px solid #2a2a45; background: #0f0f1a; color: #e0e0e0; font-size: 0.95rem;
	}
	.team-name-input:focus { outline: none; border-color: #4f46e5; }
	.team-actions { display: flex; align-items: center; gap: 0.5rem; }
	.save-status { font-size: 0.75rem; color: #4ade80; }
	.save-status.saving { color: #facc15; }
	.save-status.unsaved { color: #f87171; }

	.btn-sm {
		padding: 0.3rem 0.65rem; border-radius: 5px; border: 1px solid #2a2a45;
		background: transparent; color: #ccc; cursor: pointer; font-size: 0.78rem;
	}
	.btn-sm:hover:not(:disabled) { border-color: #555; color: #fff; }
	.btn-sm:disabled { opacity: 0.4; cursor: not-allowed; }
	.btn-sm.danger:hover { border-color: #7f1d1d; color: #f87171; }

	/* Slot grid */
	.slots-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 0.875rem; }

	.slot-card {
		background: #12121f; border: 1px solid #1e1e35; border-radius: 10px;
		padding: 0.875rem; display: flex; flex-direction: column; gap: 0.625rem;
	}
	.slot-number { font-size: 0.7rem; color: #555; text-transform: uppercase; letter-spacing: 0.06em; }

	.field-group { display: flex; flex-direction: column; gap: 0.3rem; }
	.field-label { font-size: 0.72rem; color: #666; }
	.field-input, .field-select {
		padding: 0.4rem 0.55rem; border-radius: 6px; border: 1px solid #2a2a45;
		background: #0f0f1a; color: #e0e0e0; font-size: 0.82rem; width: 100%;
	}
	.field-input:focus, .field-select:focus { outline: none; border-color: #4f46e5; }
	.field-select { cursor: pointer; }

	/* Autocomplete */
	.autocomplete { position: relative; }
	.autocomplete-input-row { display: flex; gap: 0.35rem; }
	.autocomplete-input-row .field-input { flex: 1; }
	.clear-btn {
		background: transparent; border: 1px solid #2a2a45; border-radius: 5px;
		color: #666; cursor: pointer; padding: 0 0.4rem; font-size: 0.75rem;
	}
	.clear-btn:hover { color: #f87171; border-color: #7f1d1d; }

	.dropdown {
		position: absolute; top: calc(100% + 3px); left: 0; right: 0; z-index: 20;
		background: #1a1a2e; border: 1px solid #2a2a45; border-radius: 7px;
		list-style: none; overflow: hidden; max-height: 200px; overflow-y: auto;
	}
	.dropdown li button {
		width: 100%; display: flex; align-items: center; justify-content: space-between;
		padding: 0.45rem 0.65rem; background: transparent; border: none; color: #ccc;
		cursor: pointer; font-size: 0.82rem; text-align: left;
	}
	.dropdown li button:hover { background: #2a2a45; color: #fff; }

	.types-inline { display: flex; gap: 3px; }
	.type-pip {
		padding: 1px 5px; border-radius: 8px; font-size: 0.65rem;
		font-weight: 600; color: #fff; text-transform: capitalize;
	}

	.types-row { display: flex; gap: 4px; margin-top: 2px; }
	.type-badge {
		display: inline-block; padding: 1px 8px; border-radius: 10px;
		font-size: 0.72rem; font-weight: 600; color: #fff; text-transform: capitalize;
	}

	.moves-list { display: flex; flex-direction: column; gap: 0.3rem; }
	.move-select { font-size: 0.78rem; }

	.p-sm { padding: 0.5rem 0.875rem; }
	.muted { color: #666; font-size: 0.85rem; }
</style>
