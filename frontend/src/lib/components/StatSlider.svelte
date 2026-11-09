<script lang="ts">
	import { TRAINING_POINTS_STAT_CAP } from '$lib/stats.js';

	let {
		label,
		color,
		base,
		value,
		remaining,
		onChange,
	}: {
		label: string;
		color: string;
		base: number;
		value: number;
		remaining: number; // max this stat could hold given the global 66pt budget (already includes this stat's own current points)
		onChange: (v: number) => void;
	} = $props();

	const sliderMax = $derived(Math.min(TRAINING_POINTS_STAT_CAP, remaining));
	const fillPercent = $derived(Math.min(100, (value / TRAINING_POINTS_STAT_CAP) * 100));

	function clamp(v: number) {
		if (Number.isNaN(v)) return 0;
		return Math.max(0, Math.min(sliderMax, Math.round(v)));
	}

	function dec() { onChange(clamp(value - 1)); }
	function inc() { onChange(clamp(value + 1)); }
	function reset() { onChange(0); }
	function max() { onChange(sliderMax); }
	function onNumberInput(e: Event) { onChange(clamp(Number((e.target as HTMLInputElement).value))); }

	// Native range input, always min=0/max=32 so the thumb position and drag
	// distance are always 1:1 with the fixed 32-point scale (2 points sits near
	// the left edge, matching where the visible thumb actually is). The budget
	// stop is enforced by writing the DOM's own .value property back inside the
	// same 'input' event -- synchronous, same tick, no framework round-trip --
	// so the thumb can't be dragged past sliderMax even mid-drag.
	function onSliderInput(e: Event) {
		const el = e.currentTarget as HTMLInputElement;
		const raw = Number(el.value);
		const clamped = clamp(raw);
		if (clamped !== raw) el.value = String(clamped);
		onChange(clamped);
	}
</script>

<div class="stat-slider">
	<div class="stat-slider-header">
		<span class="stat-label" style="color:{color}">{label}</span>
		<span class="stat-readout" style="color:{color}">{base + value} <span class="stat-delta">(+{value})</span></span>
	</div>
	<div class="stat-slider-row">
		<input
			class="stat-range"
			type="range"
			min="0"
			max={TRAINING_POINTS_STAT_CAP}
			step="1"
			value={clamp(value)}
			oninput={onSliderInput}
			style="--fill: {fillPercent}%; --color: {color}"
		/>

		<div class="stat-controls">
			<button class="stat-btn" onclick={dec} disabled={value <= 0} title="-1">-</button>
			<input
				class="stat-input"
				type="number"
				min="0"
				max={sliderMax}
				value={value}
				onchange={onNumberInput}
			/>
			<button class="stat-btn" onclick={inc} disabled={value >= sliderMax} title="+1">+</button>
			<button class="stat-action" onclick={reset} disabled={value === 0}>Reset</button>
			<button class="stat-action" onclick={max} disabled={value >= sliderMax}>Max</button>
		</div>
	</div>
</div>

<style>
	.stat-slider { display: flex; flex-direction: column; gap: 0.15rem; }
	.stat-slider-header { display: flex; justify-content: space-between; align-items: baseline; }
	.stat-label { font-size: 0.75rem; font-weight: 600; }
	.stat-readout { font-size: 0.85rem; font-weight: 600; }
	.stat-delta { font-size: 0.72rem; color: #999; font-weight: 400; }

	.stat-slider-row { display: flex; align-items: center; gap: 0.5rem; }

	.stat-range {
		flex: 1; height: 16px; margin: 0; cursor: pointer;
		-webkit-appearance: none; appearance: none; background: transparent;
	}
	/* Track: filled portion up to --fill, rest is the base track color */
	.stat-range::-webkit-slider-runnable-track {
		height: 4px; border-radius: 4px;
		background: linear-gradient(to right, var(--color) 0%, var(--color) var(--fill), #2a2a45 var(--fill), #2a2a45 100%);
	}
	.stat-range::-moz-range-track {
		height: 4px; border-radius: 4px; background: #2a2a45;
	}
	.stat-range::-moz-range-progress {
		height: 4px; border-radius: 4px; background: var(--color);
	}
	.stat-range::-webkit-slider-thumb {
		-webkit-appearance: none; appearance: none; margin-top: -5px;
		width: 14px; height: 14px; border-radius: 50%;
		background: var(--color); border: 2px solid #e0e0e0; cursor: pointer;
	}
	.stat-range::-moz-range-thumb {
		width: 14px; height: 14px; border-radius: 50%;
		background: var(--color); border: 2px solid #e0e0e0; cursor: pointer;
	}
	.stat-range:focus { outline: none; }
	.stat-range:focus-visible::-webkit-slider-thumb { outline: 2px solid #4f46e5; outline-offset: 2px; }
	.stat-range:focus-visible::-moz-range-thumb { outline: 2px solid #4f46e5; outline-offset: 2px; }

	.stat-controls { display: flex; align-items: center; gap: 0.3rem; flex-shrink: 0; }
	.stat-btn {
		width: 20px; height: 20px; border-radius: 5px; border: 1px solid #2a2a45;
		background: #0f0f1a; color: #ccc; cursor: pointer; font-size: 0.8rem;
		display: flex; align-items: center; justify-content: center; flex-shrink: 0; line-height: 1;
	}
	.stat-btn:hover:not(:disabled) { border-color: #555; color: #fff; }
	.stat-btn:disabled { opacity: 0.35; cursor: not-allowed; }

	.stat-input {
		width: 36px; padding: 0.1rem 0.2rem; border-radius: 5px; border: 1px solid #2a2a45;
		background: #0f0f1a; color: #e0e0e0; font-size: 0.75rem; text-align: center;
	}
	.stat-input:focus { outline: none; border-color: #4f46e5; }
	/* Hide native number spinners -- the -/+ buttons already cover that */
	.stat-input::-webkit-outer-spin-button, .stat-input::-webkit-inner-spin-button { -webkit-appearance: none; margin: 0; }
	.stat-input { appearance: textfield; -moz-appearance: textfield; }

	.stat-action {
		padding: 0.15rem 0.4rem; border-radius: 4px; border: 1px solid #2a2a45;
		background: transparent; color: #999; cursor: pointer; font-size: 0.65rem; flex-shrink: 0;
	}
	.stat-action:hover:not(:disabled) { border-color: #555; color: #fff; }
	.stat-action:disabled { opacity: 0.35; cursor: not-allowed; }
</style>
