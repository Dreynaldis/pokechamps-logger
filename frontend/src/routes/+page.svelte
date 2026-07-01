<script lang="ts">
	import { onMount } from 'svelte';

	const API = 'http://localhost:8080/api/v1';

	type PokemonSummary = {
		id: string;
		name: string;
		display_name: string;
		types: string[];
		sprite_url: string;
	};

	type MoveDetail = {
		name: string;
		display_name: string;
		type: string;
		category: string;
		power: number | null;
		accuracy: number | null;
		pp: number;
		priority: number;
		has_secondary_effect: boolean;
		is_pivot: boolean;
	};

	type AbilityDetail = {
		slot: number;
		is_hidden: boolean;
		name: string;
		display_name: string;
		short_effect: string;
		grants_immunity_type?: string;
	};

	type PokemonDetail = PokemonSummary & {
		base_stats: { hp: number; attack: number; defense: number; sp_atk: number; sp_def: number; speed: number };
		abilities: AbilityDetail[];
		moves: MoveDetail[];
	};

	let query = $state('');
	let list = $state<PokemonSummary[]>([]);
	let selected = $state<PokemonDetail | null>(null);
	let loading = $state(false);
	let detailLoading = $state(false);
	let error = $state('');

	let debounceTimer: ReturnType<typeof setTimeout>;

	onMount(() => fetchList(''));

	function onSearch() {
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => fetchList(query), 300);
	}

	async function fetchList(q: string) {
		loading = true;
		error = '';
		try {
			const res = await fetch(`${API}/pokemon?q=${encodeURIComponent(q)}&limit=50`);
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			list = await res.json();
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	}

	async function selectPokemon(name: string) {
		detailLoading = true;
		selected = null;
		try {
			const res = await fetch(`${API}/pokemon/${name}`);
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			selected = await res.json();
		} catch (e) {
			error = String(e);
		} finally {
			detailLoading = false;
		}
	}

	const typeColors: Record<string, string> = {
		normal: '#A8A878', fire: '#F08030', water: '#6890F0', electric: '#F8D030',
		grass: '#78C850', ice: '#98D8D8', fighting: '#C03028', poison: '#A040A0',
		ground: '#E0C068', flying: '#A890F0', psychic: '#F85888', bug: '#A8B820',
		rock: '#B8A038', ghost: '#705898', dragon: '#7038F8', dark: '#705848',
		steel: '#B8B8D0', fairy: '#EE99AC',
	};

	const statLabels: Record<string, string> = {
		hp: 'HP', attack: 'ATK', defense: 'DEF',
		sp_atk: 'Sp.Atk', sp_def: 'Sp.Def', speed: 'SPD',
	};
	const statMax = 255;
</script>

<main>
	<header>
		<h1>PokeChamps Logger</h1>
		<p class="subtitle">Phase 1 seed verification</p>
	</header>

	<div class="layout">
		<!-- Left: search + list -->
		<aside class="list-panel">
			<input
				type="search"
				placeholder="Search Pokemon..."
				bind:value={query}
				oninput={onSearch}
				class="search"
			/>

			{#if error}
				<p class="error">{error}</p>
			{/if}

			{#if loading}
				<p class="muted">Loading...</p>
			{:else}
				<p class="muted">{list.length} Pokemon</p>
				<ul class="pokemon-list">
					{#each list as p}
						<li>
							<button
								class="pokemon-row"
								class:active={selected?.name === p.name}
								onclick={() => selectPokemon(p.name)}
							>
								<span class="pname">{p.display_name}</span>
								<span class="types">
									{#each p.types as t}
										<span class="type-badge" style="background:{typeColors[t] ?? '#888'}">{t}</span>
									{/each}
								</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</aside>

		<!-- Right: detail -->
		<section class="detail-panel">
			{#if detailLoading}
				<p class="muted">Loading...</p>
			{:else if selected}
				<div class="detail">
					<h2>{selected.display_name}</h2>

					<div class="types row">
						{#each selected.types as t}
							<span class="type-badge large" style="background:{typeColors[t] ?? '#888'}">{t}</span>
						{/each}
					</div>

					<!-- Base stats -->
					<section class="card">
						<h3>Base Stats</h3>
						{#each Object.entries(statLabels) as [key, label]}
							{@const val = (selected.base_stats as Record<string, number>)[key] ?? 0}
							<div class="stat-row">
								<span class="stat-label">{label}</span>
								<span class="stat-val">{val}</span>
								<div class="stat-bar-bg">
									<div class="stat-bar" style="width:{Math.min(100, (val / statMax) * 100)}%"></div>
								</div>
							</div>
						{/each}
					</section>

					<!-- Abilities -->
					<section class="card">
						<h3>Abilities</h3>
						{#if selected.abilities.length === 0}
							<p class="muted">No abilities found</p>
						{:else}
							{#each selected.abilities as a}
								<div class="ability-row">
									<span class="ability-name">{a.display_name}</span>
									{#if a.is_hidden}<span class="tag">Hidden</span>{/if}
									{#if a.grants_immunity_type}<span class="tag immune">Immune: {a.grants_immunity_type}</span>{/if}
									{#if a.short_effect}<p class="ability-desc">{a.short_effect}</p>{/if}
								</div>
							{/each}
						{/if}
					</section>

					<!-- Moves -->
					<section class="card">
						<h3>Moves ({selected.moves.length})</h3>
						{#if selected.moves.length === 0}
							<p class="muted">No moves found</p>
						{:else}
							<div class="table-wrap">
								<table>
									<thead>
										<tr>
											<th>Move</th>
											<th>Type</th>
											<th>Cat.</th>
											<th>Pwr</th>
											<th>Acc</th>
											<th>PP</th>
											<th>Pri</th>
										</tr>
									</thead>
									<tbody>
										{#each selected.moves as m}
											<tr>
												<td>
													{m.display_name}
													{#if m.is_pivot}<span class="tag">Pivot</span>{/if}
													{#if m.has_secondary_effect}<span class="tag sec">+Eff</span>{/if}
												</td>
												<td>
													<span class="type-badge sm" style="background:{typeColors[m.type] ?? '#888'}">{m.type}</span>
												</td>
												<td>{m.category}</td>
												<td>{m.power ?? '—'}</td>
												<td>{m.accuracy ?? '—'}</td>
												<td>{m.pp}</td>
												<td>{m.priority !== 0 ? m.priority : '—'}</td>
											</tr>
										{/each}
									</tbody>
								</table>
							</div>
						{/if}
					</section>
				</div>
			{:else}
				<p class="muted placeholder">Select a Pokemon from the list to see its data.</p>
			{/if}
		</section>
	</div>
</main>

<style>
	* { box-sizing: border-box; margin: 0; padding: 0; }
	:global(body) { font-family: system-ui, sans-serif; background: #0f0f1a; color: #e0e0e0; }

	main { max-width: 1200px; margin: 0 auto; padding: 1rem; }

	header { padding: 1rem 0 1.5rem; }
	h1 { font-size: 1.5rem; font-weight: 700; }
	.subtitle { color: #888; font-size: 0.8rem; margin-top: 0.2rem; }

	.layout { display: grid; grid-template-columns: 280px 1fr; gap: 1rem; height: calc(100vh - 6rem); }

	/* List panel */
	.list-panel { display: flex; flex-direction: column; gap: 0.5rem; overflow: hidden; }
	.search {
		width: 100%; padding: 0.5rem 0.75rem; border-radius: 6px;
		border: 1px solid #333; background: #1a1a2e; color: #e0e0e0; font-size: 0.9rem;
	}
	.pokemon-list { list-style: none; overflow-y: auto; flex: 1; }
	.pokemon-row {
		width: 100%; display: flex; align-items: center; justify-content: space-between;
		padding: 0.4rem 0.5rem; border: none; background: transparent; color: #e0e0e0;
		cursor: pointer; border-radius: 4px; font-size: 0.85rem; text-align: left;
	}
	.pokemon-row:hover { background: #1e1e35; }
	.pokemon-row.active { background: #2a2a50; }
	.pname { flex: 1; }
	.types { display: flex; gap: 3px; }

	/* Detail panel */
	.detail-panel { overflow-y: auto; padding-right: 0.5rem; }
	.detail { display: flex; flex-direction: column; gap: 1rem; }
	h2 { font-size: 1.4rem; }
	.row { display: flex; gap: 0.4rem; margin-top: 0.25rem; }

	.card { background: #1a1a2e; border-radius: 8px; padding: 1rem; }
	h3 { font-size: 0.9rem; color: #aaa; text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.75rem; }

	/* Stats */
	.stat-row { display: grid; grid-template-columns: 60px 36px 1fr; gap: 0.5rem; align-items: center; margin-bottom: 0.4rem; }
	.stat-label { font-size: 0.78rem; color: #aaa; }
	.stat-val { font-size: 0.85rem; font-weight: 600; text-align: right; }
	.stat-bar-bg { background: #2a2a45; border-radius: 4px; height: 8px; }
	.stat-bar { background: #6366f1; border-radius: 4px; height: 8px; transition: width 0.3s; }

	/* Abilities */
	.ability-row { margin-bottom: 0.75rem; }
	.ability-name { font-weight: 600; font-size: 0.9rem; }
	.ability-desc { color: #aaa; font-size: 0.8rem; margin-top: 0.2rem; line-height: 1.4; }

	/* Moves table */
	.table-wrap { overflow-x: auto; }
	table { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
	th { text-align: left; padding: 0.4rem 0.5rem; color: #888; border-bottom: 1px solid #2a2a45; }
	td { padding: 0.35rem 0.5rem; border-bottom: 1px solid #1e1e35; }
	tr:last-child td { border-bottom: none; }

	/* Badges */
	.type-badge {
		display: inline-block; padding: 1px 7px; border-radius: 10px;
		font-size: 0.72rem; font-weight: 600; color: #fff; text-transform: capitalize;
	}
	.type-badge.large { padding: 3px 12px; font-size: 0.85rem; }
	.type-badge.sm { padding: 1px 5px; font-size: 0.7rem; }
	.tag {
		display: inline-block; margin-left: 4px; padding: 1px 5px; border-radius: 4px;
		font-size: 0.68rem; background: #2a2a50; color: #aaa;
	}
	.tag.immune { background: #1e3a2e; color: #6bcc8a; }
	.tag.sec { background: #3a2a1e; color: #cc9a6b; }

	.muted { color: #666; font-size: 0.85rem; padding: 0.5rem 0; }
	.placeholder { margin-top: 3rem; text-align: center; }
	.error { color: #f87171; font-size: 0.85rem; }
</style>
