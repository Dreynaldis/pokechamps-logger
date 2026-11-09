export type User = { id: string; email: string };

export type PokemonSummary = {
	id: string;
	name: string;
	display_name: string;
	types: string[];
	sprite_url: string;
};

export type AbilityRef = {
	id: string;
	name: string;
	display_name: string;
	short_effect: string;
	grants_immunity_type?: string;
};

export type ItemRef = {
	id: string;
	name: string;
	display_name: string;
	category: string;
	short_effect: string;
	sprite_url: string;
};

export type MoveRef = {
	id: string;
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

export type StatBlock = { hp: number; attack: number; defense: number; sp_atk: number; sp_def: number; speed: number };

export type PokemonDetail = PokemonSummary & {
	base_stats: StatBlock;
	abilities: AbilityRef[];
	moves: MoveRef[];
};

// Training points: 0-32 per stat, sum <= 66, added 1:1 to the base stat (ADR 009).
export type TrainingPointsJSON = StatBlock;

export const ZERO_TRAINING_POINTS: TrainingPointsJSON = {
	hp: 0, attack: 0, defense: 0, sp_atk: 0, sp_def: 0, speed: 0,
};

export type SlotMove = {
	slot: number;
	move: { id: string; name: string; display_name: string; type: string };
};

export type TeamSlot = {
	id: string;
	slot: number;
	pokemon: PokemonSummary;
	ability?: AbilityRef;
	item?: ItemRef;
	nature?: string;
	training_points: TrainingPointsJSON;
	moves: SlotMove[];
};

export type Team = {
	id: string;
	name: string;
	is_active: boolean;
	slots?: TeamSlot[];
};

// Draft types -- mutable local state before syncing to server
export type DraftMove = { slot: number; moveID: string; displayName: string };

export type DraftSlot = {
	slot: number; // 1-6
	pokemon: PokemonDetail | null;
	abilityID: string | null;
	itemID: string | null;
	nature: string | null;
	trainingPoints: TrainingPointsJSON;
	moves: Array<DraftMove | null>; // index 0-3 = move slots 1-4
};

export type DraftTeam = {
	id: string;
	name: string;
	slots: DraftSlot[];
};
