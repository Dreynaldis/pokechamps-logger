// Hand-rolled fuzzy matcher: case-insensitive subsequence match with bonuses
// for exact and prefix matches. Sufficient for our list sizes (Pokemon ~235,
// items ~148, moves <100 per Pokemon) -- no dependency needed.

export function fuzzyMatch(query: string, target: string): number | null {
	const q = query.trim().toLowerCase();
	const t = target.toLowerCase();
	if (!q) return 0;
	if (t === q) return 1000;
	if (t.startsWith(q)) return 500;
	if (t.includes(q)) return 250;

	// Subsequence match: every char of q must appear in order in t.
	let ti = 0;
	let score = 0;
	let consecutive = 0;
	for (let qi = 0; qi < q.length; qi++) {
		const idx = t.indexOf(q[qi], ti);
		if (idx === -1) return null;
		consecutive = idx === ti ? consecutive + 1 : 0;
		score += 10 + consecutive * 2 - (idx - ti);
		ti = idx + 1;
	}
	return score;
}

export function fuzzySort<T>(items: T[], query: string, key: (item: T) => string): T[] {
	if (!query.trim()) return items;
	const scored: Array<{ item: T; score: number }> = [];
	for (const item of items) {
		const score = fuzzyMatch(query, key(item));
		if (score !== null) scored.push({ item, score });
	}
	scored.sort((a, b) => b.score - a.score);
	return scored.map((s) => s.item);
}
