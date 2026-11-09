<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authState, authReady } from '$lib/auth.svelte.js';
	import { api } from '$lib/api.js';
	import { typeColors, typeSortIndex } from '$lib/typeOrder.js';
	import { effectiveStats, trainingPointsUsed, TRAINING_POINTS_TOTAL_CAP, TRAINING_POINTS_STAT_CAP, statColors, STAT_SHORT_LABELS } from '$lib/stats.js';
	import Combobox from '$lib/components/Combobox.svelte';
	import Select from '$lib/components/Select.svelte';
	import MoveTypeIcon from '$lib/components/MoveTypeIcon.svelte';
	import StatSlider from '$lib/components/StatSlider.svelte';
	import type {
		Team, DraftTeam, DraftSlot, DraftMove, PokemonSummary, PokemonDetail, ItemRef, AbilityRef, MoveRef,
		TrainingPointsJSON,
	} from '$lib/types.js';
	import { ZERO_TRAINING_POINTS } from '$lib/types.js';

	// ---------------------------------------------------------------------------
	// State
	// ---------------------------------------------------------------------------

	let teams = $state<Team[]>([]);
	let selectedTeamID = $state<string | null>(null);
	let draft = $state<DraftTeam | null>(null);
	let saveStatus = $state<'saved' | 'saving' | 'unsaved'>('saved');
	let teamsLoading = $state(true);
	let selectedSlotIdx = $state(0);

	let slotDetail = $state<Array<PokemonDetail | null>>(Array(6).fill(null));
	let allItems = $state<ItemRef[]>([]);
	let allPokemon = $state<PokemonSummary[]>([]);

	let lsTimer: ReturnType<typeof setTimeout>;
	let serverTimer: ReturnType<typeof setTimeout>;

	const NATURES = ['Hardy','Lonely','Brave','Adamant','Naughty','Bold','Docile','Relaxed',
		'Impish','Lax','Timid','Hasty','Serious','Jolly','Naive','Modest','Mild',
		'Quiet','Bashful','Rash','Calm','Gentle','Sassy','Careful','Quirky'];
	const NATURE_ITEMS = NATURES.map(n => ({ id: n, label: n }));
	const STAT_LABELS: Array<{ key: keyof TrainingPointsJSON; label: string }> = [
		{ key: 'hp', label: 'HP' }, { key: 'attack', label: 'Attack' }, { key: 'defense', label: 'Defense' },
		{ key: 'sp_atk', label: 'Sp. Atk' }, { key: 'sp_def', label: 'Sp. Def' }, { key: 'speed', label: 'Speed' },
	];
	const STAT_MAX = 255; // conventional Pokemon base-stat ceiling, matches Phase 1 verification page

	// ---------------------------------------------------------------------------
	// Boot
	// ---------------------------------------------------------------------------

	onMount(async () => {
		await authReady;
		if (!authState.user) { goto('/auth'); return; }
		await loadTeams();
		await loadItems();
		await loadPokemon();
	});

	async function loadItems() {
		const res = await api('/api/v1/items?limit=200');
		if (res.ok) allItems = await res.json();
	}

	async function loadPokemon() {
		const res = await api('/api/v1/pokemon?limit=300');
		if (res.ok) allPokemon = await res.json();
	}

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
					item_id: s.itemID ?? undefined,
					nature: s.nature ?? undefined,
					training_points: s.trainingPoints,
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
		selectedSlotIdx = 0;
		const res = await api(`/api/v1/teams/${id}`);
		if (!res.ok) return;
		await buildDraft(await res.json());
	}

	async function buildDraft(team: Team) {
		const emptySlots: DraftSlot[] = Array.from({ length: 6 }, (_, i) => ({
			slot: i + 1, pokemon: null, abilityID: null, itemID: null, nature: null,
			trainingPoints: { ...ZERO_TRAINING_POINTS },
			moves: [null, null, null, null],
		}));

		const newDetail: Array<PokemonDetail | null> = Array(6).fill(null);

		for (const s of team.slots ?? []) {
			const idx = s.slot - 1;
			const detailRes = await api(`/api/v1/pokemon/${s.pokemon.name}`);
			const detail: PokemonDetail = detailRes.ok ? await detailRes.json() : s.pokemon as unknown as PokemonDetail;
			newDetail[idx] = detail;

			const moves: Array<DraftMove | null> = [null, null, null, null];
			for (const m of s.moves) {
				moves[m.slot - 1] = { slot: m.slot, moveID: m.move.id, displayName: m.move.display_name };
			}
			emptySlots[idx] = {
				slot: s.slot, pokemon: detail,
				abilityID: s.ability?.id ?? null, itemID: s.item?.id ?? null,
				nature: s.nature ?? null,
				trainingPoints: s.training_points ?? { ...ZERO_TRAINING_POINTS },
				moves,
			};
		}

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

	async function selectPokemon(idx: number, p: PokemonSummary | null) {
		if (!p) return;
		const detailRes = await api(`/api/v1/pokemon/${p.name}`);
		if (!detailRes.ok) return;
		const detail: PokemonDetail = await detailRes.json();
		slotDetail[idx] = detail;
		draft!.slots[idx] = {
			slot: idx + 1, pokemon: detail,
			abilityID: detail.abilities[0]?.id ?? null, itemID: null, nature: null,
			trainingPoints: { ...ZERO_TRAINING_POINTS },
			moves: [null, null, null, null],
		};
		draft = { ...draft! };
		markDirty();
	}

	function clearSlot(idx: number) {
		slotDetail[idx] = null;
		draft!.slots[idx] = {
			slot: idx + 1, pokemon: null, abilityID: null, itemID: null, nature: null,
			trainingPoints: { ...ZERO_TRAINING_POINTS }, moves: [null, null, null, null],
		};
		draft = { ...draft! };
		markDirty();
	}

	function setAbility(idx: number, a: AbilityRef | null) {
		draft!.slots[idx] = { ...draft!.slots[idx], abilityID: a?.id ?? null };
		draft = { ...draft! };
		markDirty();
	}

	function setItem(idx: number, i: ItemRef | null) {
		draft!.slots[idx] = { ...draft!.slots[idx], itemID: i?.id ?? null };
		draft = { ...draft! };
		markDirty();
	}

	function setNature(idx: number, n: { id: string; label: string } | null) {
		draft!.slots[idx] = { ...draft!.slots[idx], nature: n?.id ?? null };
		draft = { ...draft! };
		markDirty();
	}

	function setMove(idx: number, moveSlot: number, m: MoveRef | null) {
		const moves = [...draft!.slots[idx].moves];
		moves[moveSlot] = m ? { slot: moveSlot + 1, moveID: m.id, displayName: m.display_name } : null;
		draft!.slots[idx] = { ...draft!.slots[idx], moves };
		draft = { ...draft! };
		markDirty();
	}

	function setTrainingPoint(idx: number, stat: keyof TrainingPointsJSON, value: number) {
		const current = draft!.slots[idx].trainingPoints;
		// Defensive clamp at the source of truth -- don't rely solely on the slider's own
		// clamping, since drag/direct-input can report values outside the computed max.
		const others = trainingPointsUsed(current) - current[stat];
		const maxAllowed = Math.min(TRAINING_POINTS_STAT_CAP, TRAINING_POINTS_TOTAL_CAP - others);
		const clamped = Math.max(0, Math.min(maxAllowed, Math.round(value)));
		const tp = { ...current, [stat]: clamped };
		draft!.slots[idx] = { ...draft!.slots[idx], trainingPoints: tp };
		draft = { ...draft! };
		markDirty();
	}

	function onTeamNameInput() {
		markDirty();
	}

	function sortedMoves(detail: PokemonDetail | null): MoveRef[] {
		if (!detail) return [];
		return [...detail.moves].sort((a, b) => {
			const ti = typeSortIndex(a.type) - typeSortIndex(b.type);
			return ti !== 0 ? ti : a.display_name.localeCompare(b.display_name);
		});
	}
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

			<!-- Master-detail: slot list (left) + slot editor (right) -->
			<div class="master-detail">
				<div class="slot-list-panel">
					{#each draft.slots as slot, idx}
						{@const detail = slotDetail[idx]}
						{@const stats = detail ? effectiveStats(detail.base_stats, slot.trainingPoints) : null}
						{@const abilityName = detail?.abilities.find(a => a.id === slot.abilityID)?.display_name}
						{@const itemName = allItems.find(i => i.id === slot.itemID)?.display_name}
						{@const moveNames = slot.moves.filter((m): m is DraftMove => m !== null).map(m => m.displayName)}
						<button
							class="slot-row"
							class:selected={selectedSlotIdx === idx}
							onclick={() => selectedSlotIdx = idx}
						>
							{#if slot.pokemon}
								<img class="sprite-md" src={slot.pokemon.sprite_url} alt="" loading="lazy" />
								<div class="slot-row-info">
									<span class="slot-row-name">{slot.pokemon.display_name}</span>
									<div class="types-inline">
										{#each slot.pokemon.types as t}
											<span class="type-pip" style="background:{typeColors[t] ?? '#888'}">{t}</span>
										{/each}
									</div>

									<div class="slot-row-meta"><span class="slot-row-meta-label">Ability:</span> {abilityName ?? '--'}</div>
									<div class="slot-row-meta"><span class="slot-row-meta-label">Item:</span> {itemName ?? '--'}</div>
									<div class="slot-row-meta">
										<span class="slot-row-meta-label">Moves:</span> {moveNames.length ? moveNames.join(', ') : '--'}
									</div>

									{#if stats}
										<div class="mini-stat-graph">
											{#each STAT_LABELS as { key }}
												<div class="mini-stat-row" title="{STAT_SHORT_LABELS[key]} {stats[key]}">
													<span class="mini-stat-label" style="color:{statColors[key]}">{STAT_SHORT_LABELS[key]}</span>
													<div class="mini-stat-bar-bg">
														<div
															class="mini-stat-bar"
															style="width:{Math.min(100, (stats[key] / STAT_MAX) * 100)}%; background:{statColors[key]}"
														></div>
													</div>
													<span class="mini-stat-val">{stats[key]}</span>
												</div>
											{/each}
										</div>
									{/if}
								</div>
							{:else}
								<span class="slot-row-empty">Slot {slot.slot} -- empty</span>
							{/if}
						</button>
					{/each}
				</div>

				<div class="slot-detail-panel">
				{#each [selectedSlotIdx] as idx (idx)}
					{@const slot = draft.slots[idx]}
					{@const detail = slotDetail[idx]}

					<div class="detail-species-row">
						<span class="field-label">Slot {slot.slot} -- Species</span>
						<div class="detail-species-picker">
							<Combobox
								items={allPokemon}
								getLabel={(p: PokemonSummary) => p.display_name}
								getValue={(p: PokemonSummary) => p.id}
								value={slot.pokemon?.id ?? null}
								onSelect={(p) => selectPokemon(idx, p)}
								placeholder="Search Pokemon..."
							>
								{#snippet item(p: PokemonSummary)}
									<img class="sprite-sm" src={p.sprite_url} alt="" loading="lazy" />
									<span class="cb-item-name">{p.display_name}</span>
									<span class="types-inline">
										{#each p.types as t}
											<span class="type-pip" style="background:{typeColors[t] ?? '#888'}">{t}</span>
										{/each}
									</span>
								{/snippet}
							</Combobox>
							{#if slot.pokemon}
								<button class="clear-btn" onclick={() => clearSlot(idx)}>✕ Clear</button>
							{/if}
						</div>
					</div>

					{#if slot.pokemon && detail}
						{@const summaryStats = effectiveStats(detail.base_stats, slot.trainingPoints)}
						<div class="detail-body">
							<div class="selected-pokemon-row">
								<img class="sprite-lg" src={slot.pokemon.sprite_url} alt={slot.pokemon.display_name} loading="lazy" />
								<div class="types-row">
									{#each slot.pokemon.types as t}
										<span class="type-badge" style="background:{typeColors[t] ?? '#888'}">{t}</span>
									{/each}
								</div>
							</div>

							<div class="stat-graph">
								{#each STAT_LABELS as { key, label }}
									<div class="stat-graph-row">
										<span class="stat-graph-label" style="color:{statColors[key]}">{label}</span>
										<span class="stat-graph-val">{summaryStats[key]}</span>
										<div class="stat-graph-bar-bg">
											<div
												class="stat-graph-bar"
												style="width:{Math.min(100, (summaryStats[key] / STAT_MAX) * 100)}%; background:{statColors[key]}"
											></div>
										</div>
									</div>
								{/each}
								<div class="stat-graph-total">
									Total: {STAT_LABELS.reduce((sum, { key }) => sum + summaryStats[key], 0)}
								</div>
							</div>

							<div class="field-group">
								<span class="field-label">Ability</span>
								<Select
									items={detail.abilities}
									getLabel={(a: AbilityRef) => a.display_name}
									getValue={(a: AbilityRef) => a.id}
									value={slot.abilityID}
									onSelect={(a) => setAbility(idx, a)}
									placeholder="-- none --"
								/>
							</div>

							<div class="field-group">
								<span class="field-label">Held Item</span>
								<Combobox
									items={allItems}
									getLabel={(i: ItemRef) => i.display_name}
									getValue={(i: ItemRef) => i.id}
									value={slot.itemID}
									onSelect={(i) => setItem(idx, i)}
									placeholder="Search items..."
								>
									{#snippet item(i: ItemRef)}
										<img class="sprite-sm" src={i.sprite_url} alt="" loading="lazy" />
										<span class="cb-item-name">{i.display_name}</span>
									{/snippet}
								</Combobox>
							</div>

							<div class="field-group">
								<span class="field-label">Nature</span>
								<Select
									items={NATURE_ITEMS}
									getLabel={(n) => n.label}
									getValue={(n) => n.id}
									value={slot.nature}
									onSelect={(n) => setNature(idx, n)}
									placeholder="-- none --"
								/>
							</div>

							<div class="field-group">
								<span class="field-label">Moves</span>
								<div class="moves-list">
									{#each [0, 1, 2, 3] as mi}
										<Combobox
											items={sortedMoves(detail)}
											getLabel={(m: MoveRef) => m.display_name}
											getValue={(m: MoveRef) => m.id}
											value={slot.moves[mi]?.moveID ?? null}
											onSelect={(m) => setMove(idx, mi, m)}
											placeholder={`Move ${mi + 1}`}
										>
											{#snippet item(m: MoveRef)}
												<MoveTypeIcon type={m.type} />
												<span class="cb-item-name">{m.display_name}</span>
											{/snippet}
										</Combobox>
									{/each}
								</div>
							</div>

							<div class="field-group">
								<span class="field-label">
									Training Points ({trainingPointsUsed(slot.trainingPoints)}/{TRAINING_POINTS_TOTAL_CAP})
								</span>
								<div class="stat-sliders">
									{#each STAT_LABELS as { key, label }}
										{@const used = trainingPointsUsed(slot.trainingPoints)}
										{@const remaining = TRAINING_POINTS_TOTAL_CAP - used + slot.trainingPoints[key]}
										<StatSlider
											{label}
											color={statColors[key]}
											base={detail.base_stats[key]}
											value={slot.trainingPoints[key]}
											{remaining}
											onChange={(v) => setTrainingPoint(idx, key, v)}
										/>
									{/each}
								</div>
							</div>
						</div>
					{:else}
						<div class="empty-state small">Pick a Pokemon for this slot.</div>
					{/if}
				{/each}
				</div>
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
	.builder { overflow: hidden; padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem; min-height: 0; }

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

	/* Master-detail */
	.master-detail { display: grid; grid-template-columns: 280px 1fr; gap: 0.875rem; flex: 1; min-height: 0; }

	.slot-list-panel {
		display: flex; flex-direction: column; gap: 0.5rem; overflow-y: auto;
		border-right: 1px solid #1e1e35; padding-right: 0.875rem;
	}
	.slot-row {
		display: flex; align-items: flex-start; gap: 0.6rem; text-align: left;
		background: #12121f; border: 1px solid #1e1e35; border-radius: 10px;
		padding: 0.6rem 0.75rem; cursor: pointer; color: #ccc;
	}
	.slot-row:hover { border-color: #555; }
	.slot-row.selected { border-color: #4f46e5; background: #161628; }
	.slot-row-info { display: flex; flex-direction: column; gap: 2px; min-width: 0; flex: 1; }
	.slot-row-name { font-size: 0.85rem; color: #e0e0e0; font-weight: 600; }

	.slot-row-meta {
		font-size: 0.68rem; color: #999; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
	}
	.slot-row-meta-label { color: #666; }

	.mini-stat-graph { display: flex; flex-direction: column; gap: 1px; margin-top: 2px; }
	.mini-stat-row { display: grid; grid-template-columns: 20px 1fr 22px; align-items: center; gap: 4px; }
	.mini-stat-label { font-size: 0.55rem; font-weight: 700; line-height: 1; }
	.mini-stat-bar-bg { background: #2a2a45; border-radius: 2px; height: 3px; }
	.mini-stat-bar { border-radius: 2px; height: 3px; transition: width 0.3s ease-out; }
	.mini-stat-val { font-size: 0.6rem; color: #999; text-align: right; }
	.slot-row-empty { font-size: 0.8rem; color: #555; }

	.slot-detail-panel {
		display: flex; flex-direction: column; gap: 0.875rem; overflow-y: auto; min-height: 0;
	}
	.detail-species-row { display: flex; flex-direction: column; gap: 0.3rem; }
	.detail-species-picker { display: flex; gap: 0.35rem; align-items: center; }
	.detail-species-picker :global(.cb-input) { flex: 1; }

	.detail-body {
		display: flex; flex-direction: column; gap: 0.75rem;
		background: #12121f; border: 1px solid #1e1e35; border-radius: 10px; padding: 0.875rem;
	}

	.field-group { display: flex; flex-direction: column; gap: 0.3rem; }
	.field-label { font-size: 0.72rem; color: #666; }

	.clear-btn {
		background: transparent; border: 1px solid #2a2a45; border-radius: 5px;
		color: #666; cursor: pointer; padding: 0.3rem 0.5rem; font-size: 0.75rem; white-space: nowrap;
	}
	.clear-btn:hover { color: #f87171; border-color: #7f1d1d; }

	.cb-item-name { flex: 1; }
	.sprite-sm { width: 24px; height: 24px; object-fit: contain; flex-shrink: 0; }
	.sprite-md { width: 44px; height: 44px; object-fit: contain; flex-shrink: 0; }
	.sprite-lg { width: 48px; height: 48px; object-fit: contain; flex-shrink: 0; }
	.selected-pokemon-row { display: flex; align-items: center; gap: 0.5rem; }

	/* Stat graph -- horizontal bars, colored per stat, same pattern as Phase 1 verification page */
	.stat-graph {
		display: flex; flex-direction: column; gap: 0.3rem;
		background: #0f0f1a; border: 1px solid #1e1e35; border-radius: 8px; padding: 0.75rem 0.875rem;
	}
	.stat-graph-row { display: grid; grid-template-columns: 60px 32px 1fr; align-items: center; gap: 0.5rem; }
	.stat-graph-label { font-size: 0.72rem; font-weight: 600; }
	.stat-graph-val { font-size: 0.78rem; font-weight: 600; color: #e0e0e0; text-align: right; }
	.stat-graph-bar-bg { background: #2a2a45; border-radius: 4px; height: 8px; }
	.stat-graph-bar { border-radius: 4px; height: 8px; transition: width 0.3s ease-out; }
	.stat-graph-total {
		margin-top: 0.2rem; padding-top: 0.4rem; border-top: 1px solid #1e1e35;
		font-size: 0.75rem; color: #999; text-align: right;
	}

	.types-inline { display: flex; gap: 3px; }
	.type-pip {
		padding: 1px 5px; border-radius: 8px; font-size: 0.65rem;
		font-weight: 600; color: #fff; text-transform: capitalize;
		text-shadow: 0 0 2px rgba(0, 0, 0, 0.85), 0 1px 1px rgba(0, 0, 0, 0.85);
	}

	.types-row { display: flex; gap: 4px; }
	.type-badge {
		display: inline-block; padding: 1px 8px; border-radius: 10px;
		font-size: 0.72rem; font-weight: 600; color: #fff; text-transform: capitalize;
		text-shadow: 0 0 2px rgba(0, 0, 0, 0.85), 0 1px 1px rgba(0, 0, 0, 0.85);
	}

	.moves-list { display: flex; flex-direction: column; gap: 0.3rem; }

	.stat-sliders { display: flex; flex-direction: column; gap: 0.25rem; }

	.empty-state.small { height: auto; padding: 2rem 0; }

	.p-sm { padding: 0.5rem 0.875rem; }
	.muted { color: #666; font-size: 0.85rem; }

	/* Responsive: below 900px, stack slot list above the detail panel */
	@media (max-width: 900px) {
		.master-detail { grid-template-columns: 1fr; }
		.slot-list-panel {
			flex-direction: row; overflow-x: auto; overflow-y: hidden;
			border-right: none; border-bottom: 1px solid #1e1e35; padding-right: 0; padding-bottom: 0.5rem;
		}
		.slot-row { flex-shrink: 0; min-width: 200px; }
	}
</style>
