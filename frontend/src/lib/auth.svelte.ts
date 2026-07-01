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
