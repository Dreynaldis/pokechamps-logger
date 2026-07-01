<script lang="ts">
	import { onMount } from 'svelte';
	import { loginWithToken } from '$lib/auth.svelte.js';
	import { api, setApiToken } from '$lib/api.js';

	let status = $state<'loading' | 'error'>('loading');

	onMount(async () => {
		// OAuth backend redirects here with: /auth/callback#access_token=<jwt>
		const hash = window.location.hash.slice(1);
		const params = new URLSearchParams(hash);
		const token = params.get('access_token');

		if (!token) {
			const err = params.get('error');
			if (err) {
				status = 'error';
				return;
			}
		}

		if (token) {
			setApiToken(token);
			const res = await api('/api/v1/auth/me');
			if (res.ok) {
				loginWithToken(token, await res.json());
				// Clear the token from the URL before navigating
				history.replaceState(null, '', window.location.pathname);
				window.location.href = '/teams';
				return;
			}
		}

		status = 'error';
	});
</script>

{#if status === 'loading'}
	<div class="center"><p class="muted">Signing you in...</p></div>
{:else}
	<div class="center">
		<p class="error">Sign-in failed. <a href="/auth">Try again</a></p>
	</div>
{/if}

<style>
	.center { display: flex; align-items: center; justify-content: center; min-height: calc(100vh - 52px); }
	.muted { color: #888; }
	.error { color: #f87171; }
	a { color: #a5b4fc; }
</style>
