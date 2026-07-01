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

export type PokemonDetail = PokemonSummary & {
	base_stats: { hp: number; attack: number; defense: number; sp_atk: number; sp_def: number; speed: number };
	abilities: AbilityRef[];
	moves: MoveRef[];
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
	item?: { id: string; name: string; display_name: string };
	nature?: string;
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
	nature: string | null;
	moves: Array<DraftMove | null>; // index 0-3 = move slots 1-4
};

export type DraftTeam = {
	id: string;
	name: string;
	slots: DraftSlot[];
};
