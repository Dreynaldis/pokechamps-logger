import { setApiToken } from './api.js';
import type { User } from './types.js';

// Svelte 5 reactive auth state. Shared across all components via import.
export const authState = $state<{ user: User | null; loading: boolean }>({
	user: null,
	loading: true,
});

export function loginWithToken(token: string, user: User) {
	setApiToken(token);
	authState.user = user;
	authState.loading = false;
}

export function clearAuth() {
	setApiToken(null);
	authState.user = null;
	authState.loading = false;
}

// Resolves once the layout's initial session-restore attempt (success or
// failure) has finished. Pages that gate on authState.user must await this
// first, otherwise they race the restore and bounce a logged-in user to /auth.
let resolveAuthReady: () => void;
export const authReady = new Promise<void>((resolve) => { resolveAuthReady = resolve; });
export function markAuthReady() {
	resolveAuthReady();
}
