<script lang="ts" generics="T">
	import { Combobox } from 'bits-ui';
	import { fuzzySort } from '$lib/fuzzy.js';
	import type { Snippet } from 'svelte';

	let {
		items,
		getLabel,
		getValue,
		value = $bindable(null),
		onSelect,
		placeholder = 'Search...',
		item: itemSnippet,
	}: {
		items: T[];
		getLabel: (item: T) => string;
		getValue: (item: T) => string;
		value?: string | null;
		onSelect?: (item: T | null) => void;
		placeholder?: string;
		item?: Snippet<[T]>;
	} = $props();

	let query = $state('');
	let open = $state(false);

	const filtered = $derived(query ? fuzzySort(items, query, getLabel) : items);
	const selectedItem = $derived(items.find((i) => getValue(i) === value) ?? null);
	const inputValue = $derived(open ? query : (selectedItem ? getLabel(selectedItem) : ''));

	function handleValueChange(v: string) {
		value = v || null;
		const found = items.find((i) => getValue(i) === v) ?? null;
		onSelect?.(found);
		query = '';
	}
</script>

<Combobox.Root type="single" value={value ?? ''} onValueChange={handleValueChange} bind:open inputValue={inputValue}>
	<Combobox.Input
		class="cb-input"
		{placeholder}
		oninput={(e) => { query = e.currentTarget.value; open = true; }}
		onfocus={() => { open = true; }}
		onclick={() => { open = true; }}
	/>
	<Combobox.Portal>
		<Combobox.Content class="cb-content" sideOffset={4} style="width: var(--bits-combobox-anchor-width)">
			<Combobox.Viewport class="cb-viewport">
				{#each filtered as it (getValue(it))}
					<Combobox.Item value={getValue(it)} label={getLabel(it)} class="cb-item">
						{#if itemSnippet}
							{@render itemSnippet(it)}
						{:else}
							<span>{getLabel(it)}</span>
						{/if}
					</Combobox.Item>
				{:else}
					<div class="cb-empty">No results</div>
				{/each}
			</Combobox.Viewport>
		</Combobox.Content>
	</Combobox.Portal>
</Combobox.Root>

<style>
	:global(.cb-input) {
		padding: 0.4rem 0.55rem; border-radius: 6px; border: 1px solid #2a2a45;
		background: #0f0f1a; color: #e0e0e0; font-size: 0.82rem; width: 100%;
	}
	:global(.cb-input:focus) { outline: none; border-color: #4f46e5; }

	:global(.cb-content) {
		background: #1a1a2e; border: 1px solid #2a2a45; border-radius: 7px;
		max-height: 240px; overflow: hidden; z-index: 50;
	}
	:global(.cb-viewport) { overflow-y: auto; max-height: 240px; }

	:global(.cb-item) {
		display: flex; align-items: center; gap: 0.4rem;
		padding: 0.45rem 0.65rem; color: #ccc; cursor: pointer; font-size: 0.82rem;
	}
	:global(.cb-item[data-highlighted]) { background: #2a2a45; color: #fff; }

	:global(.cb-empty) { padding: 0.6rem 0.65rem; color: #666; font-size: 0.8rem; }
</style>
