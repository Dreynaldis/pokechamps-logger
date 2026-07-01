<script lang="ts">
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';
	import { authState, loginWithToken, clearAuth } from '$lib/auth.svelte.js';
	import { api, setApiToken, BASE } from '$lib/api.js';

	let { children }: { children: Snippet } = $props();

	onMount(async () => {
		// On every page load, try to get a fresh access token from the refresh cookie.
		// If the cookie is absent or expired, the user is logged out.
		try {
			const ref = await fetch(`${BASE}/auth/refresh`, { method: 'POST', credentials: 'include' });
			if (ref.ok) {
				const { access_token } = await ref.json();
				setApiToken(access_token);
				const me = await api('/api/v1/auth/me');
				if (me.ok) {
					loginWithToken(access_token, await me.json());
					return;
				}
			}
		} catch {}
		clearAuth();
	});

	async function handleLogout() {
		await api('/auth/logout', { method: 'POST' });
		clearAuth();
		window.location.href = '/';
	}
</script>

<div class="app">
	<nav class="navbar">
		<a href="/" class="logo">PokeChamps</a>

		<div class="nav-links">
			<a href="/">Pokédex</a>
			<a href="/teams">Teams</a>
		</div>

		<div class="nav-auth">
			{#if authState.loading}
				<span class="muted">...</span>
			{:else if authState.user}
				<span class="nav-email">{authState.user.email}</span>
				<button class="btn-ghost" onclick={handleLogout}>Logout</button>
			{:else}
				<a href="/auth" class="btn-primary">Sign in</a>
			{/if}
		</div>
	</nav>

	<main>
		{@render children()}
	</main>
</div>

<style>
	:global(*) { box-sizing: border-box; margin: 0; padding: 0; }
	:global(body) { font-family: system-ui, sans-serif; background: #0f0f1a; color: #e0e0e0; }
	:global(a) { color: inherit; text-decoration: none; }

	.app { display: flex; flex-direction: column; min-height: 100vh; }

	.navbar {
		display: flex; align-items: center; gap: 1.5rem;
		padding: 0 1.5rem; height: 52px;
		background: #12121f; border-bottom: 1px solid #1e1e35;
		position: sticky; top: 0; z-index: 10;
	}

	.logo { font-weight: 700; font-size: 1rem; color: #a5b4fc; letter-spacing: 0.02em; }

	.nav-links { display: flex; gap: 1rem; flex: 1; }
	.nav-links a { font-size: 0.875rem; color: #aaa; padding: 0.25rem 0; }
	.nav-links a:hover { color: #e0e0e0; }

	.nav-auth { display: flex; align-items: center; gap: 0.75rem; }
	.nav-email { font-size: 0.8rem; color: #888; }

	.btn-primary {
		padding: 0.3rem 0.85rem; border-radius: 6px; font-size: 0.8rem;
		background: #4f46e5; color: #fff; border: none; cursor: pointer;
	}
	.btn-primary:hover { background: #4338ca; }

	.btn-ghost {
		padding: 0.3rem 0.75rem; border-radius: 6px; font-size: 0.8rem;
		background: transparent; color: #aaa; border: 1px solid #333; cursor: pointer;
	}
	.btn-ghost:hover { color: #e0e0e0; border-color: #555; }

	.muted { color: #666; font-size: 0.85rem; }

	main { flex: 1; }
</style>
