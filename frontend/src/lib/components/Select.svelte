<script lang="ts" generics="T">
	import { Select } from 'bits-ui';
	import type { Snippet } from 'svelte';

	let {
		items,
		getLabel,
		getValue,
		value = $bindable(null),
		onSelect,
		placeholder = '-- none --',
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

	const selectedItem = $derived(items.find((i) => getValue(i) === value) ?? null);

	function handleValueChange(v: string) {
		value = v || null;
		onSelect?.(items.find((i) => getValue(i) === v) ?? null);
	}
</script>

<Select.Root type="single" value={value ?? ''} onValueChange={handleValueChange}>
	<Select.Trigger class="sel-trigger">
		{selectedItem ? getLabel(selectedItem) : placeholder}
	</Select.Trigger>
	<Select.Portal>
		<Select.Content class="sel-content" sideOffset={4} style="width: var(--bits-select-anchor-width)">
			<Select.Viewport class="sel-viewport">
				<Select.Item value="" label={placeholder} class="sel-item">
					<span class="sel-item-muted">{placeholder}</span>
				</Select.Item>
				{#each items as it (getValue(it))}
					<Select.Item value={getValue(it)} label={getLabel(it)} class="sel-item">
						{#if itemSnippet}
							{@render itemSnippet(it)}
						{:else}
							<span>{getLabel(it)}</span>
						{/if}
					</Select.Item>
				{/each}
			</Select.Viewport>
		</Select.Content>
	</Select.Portal>
</Select.Root>

<style>
	:global(.sel-trigger) {
		padding: 0.4rem 0.55rem; border-radius: 6px; border: 1px solid #2a2a45;
		background: #0f0f1a; color: #e0e0e0; font-size: 0.82rem; width: 100%;
		text-align: left; cursor: pointer;
	}
	:global(.sel-trigger:focus) { outline: none; border-color: #4f46e5; }

	:global(.sel-content) {
		background: #1a1a2e; border: 1px solid #2a2a45; border-radius: 7px;
		max-height: 240px; overflow: hidden; z-index: 50;
	}
	:global(.sel-viewport) { overflow-y: auto; max-height: 240px; }

	:global(.sel-item) {
		display: flex; align-items: center; gap: 0.4rem;
		padding: 0.45rem 0.65rem; color: #ccc; cursor: pointer; font-size: 0.82rem;
	}
	:global(.sel-item[data-highlighted]) { background: #2a2a45; color: #fff; }
	:global(.sel-item-muted) { color: #666; }
</style>
