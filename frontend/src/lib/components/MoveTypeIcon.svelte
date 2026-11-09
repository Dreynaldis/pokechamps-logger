<script lang="ts">
	import { typeColors } from '$lib/typeOrder.js';

	let { type, size = 'sm' }: { type: string; size?: 'sm' | 'md' } = $props();
</script>

<span class="move-type-icon {size}" style="background:{typeColors[type] ?? '#888'}">{type}</span>

<style>
	.move-type-icon {
		display: inline-flex; align-items: center; justify-content: center;
		border-radius: 8px; font-weight: 600; color: #fff; text-transform: capitalize;
		text-shadow: 0 0 2px rgba(0, 0, 0, 0.85), 0 1px 1px rgba(0, 0, 0, 0.85);
		flex-shrink: 0; white-space: nowrap;
	}
	.move-type-icon.sm { padding: 1px 6px; font-size: 0.65rem; }
	.move-type-icon.md { padding: 2px 8px; font-size: 0.72rem; }
</style>
