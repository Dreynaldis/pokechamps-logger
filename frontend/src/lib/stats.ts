import type { StatBlock, TrainingPointsJSON } from './types.js';

// Training points add 1:1 to the base stat, no divisor (see ADR 009).
export function effectiveStats(base: StatBlock, tp: TrainingPointsJSON): StatBlock {
	return {
		hp: base.hp + tp.hp,
		attack: base.attack + tp.attack,
		defense: base.defense + tp.defense,
		sp_atk: base.sp_atk + tp.sp_atk,
		sp_def: base.sp_def + tp.sp_def,
		speed: base.speed + tp.speed,
	};
}

export const TRAINING_POINTS_TOTAL_CAP = 66;
export const TRAINING_POINTS_STAT_CAP = 32;

export function trainingPointsUsed(tp: TrainingPointsJSON): number {
	return tp.hp + tp.attack + tp.defense + tp.sp_atk + tp.sp_def + tp.speed;
}

// Per-stat accent colors for stat summaries/readouts (distinct hues, tuned for dark backgrounds).
export const statColors: Record<keyof StatBlock, string> = {
	hp: '#22c55e',
	attack: '#ef4444',
	defense: '#eab308',
	sp_atk: '#3b82f6',
	sp_def: '#a855f7',
	speed: '#ec4899',
};

export const STAT_SHORT_LABELS: Record<keyof StatBlock, string> = {
	hp: 'HP', attack: 'Atk', defense: 'Def', sp_atk: 'SpA', sp_def: 'SpD', speed: 'Spe',
};
