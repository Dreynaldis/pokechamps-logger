<script lang="ts">
	import { goto } from '$app/navigation';
	import { loginWithToken } from '$lib/auth.svelte.js';
	import { BASE } from '$lib/api.js';

	let tab = $state<'login' | 'register'>('login');
	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function submit() {
		error = '';
		loading = true;
		try {
			const path = tab === 'login' ? '/auth/login' : '/auth/register';
			const res = await fetch(BASE + path, {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, password }),
			});
			const data = await res.json();
			if (!res.ok) {
				error = data.error ?? 'Something went wrong';
				return;
			}
			loginWithToken(data.access_token, data.user);
			goto('/teams');
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') submit();
	}
</script>

<div class="auth-page">
	<div class="auth-card">
		<h1 class="title">PokeChamps Logger</h1>

		<div class="tabs">
			<button class:active={tab === 'login'} onclick={() => { tab = 'login'; error = ''; }}>
				Sign in
			</button>
			<button class:active={tab === 'register'} onclick={() => { tab = 'register'; error = ''; }}>
				Create account
			</button>
		</div>

		<div class="form">
			<label>
				Email
				<input type="email" bind:value={email} onkeydown={onKeydown} placeholder="you@example.com" autocomplete="email" />
			</label>

			<label>
				Password
				<input type="password" bind:value={password} onkeydown={onKeydown} placeholder={tab === 'register' ? 'Min 8 characters' : ''} autocomplete={tab === 'login' ? 'current-password' : 'new-password'} />
			</label>

			{#if error}
				<p class="error">{error}</p>
			{/if}

			<button class="submit" onclick={submit} disabled={loading || !email || !password}>
				{loading ? 'Please wait...' : tab === 'login' ? 'Sign in' : 'Create account'}
			</button>
		</div>

		<div class="divider"><span>or</span></div>

		<div class="oauth">
			<a href="http://localhost:8080/auth/google" class="oauth-btn google">
				Continue with Google
			</a>
			<a href="http://localhost:8080/auth/discord" class="oauth-btn discord">
				Continue with Discord
			</a>
		</div>
	</div>
</div>

<style>
	.auth-page {
		display: flex; align-items: center; justify-content: center;
		min-height: calc(100vh - 52px); padding: 2rem;
	}

	.auth-card {
		width: 100%; max-width: 380px;
		background: #12121f; border: 1px solid #1e1e35; border-radius: 12px;
		padding: 2rem;
	}

	.title { font-size: 1.25rem; font-weight: 700; color: #a5b4fc; margin-bottom: 1.5rem; text-align: center; }

	.tabs { display: flex; gap: 0; border: 1px solid #2a2a45; border-radius: 8px; overflow: hidden; margin-bottom: 1.5rem; }
	.tabs button {
		flex: 1; padding: 0.5rem; background: transparent; color: #888;
		border: none; cursor: pointer; font-size: 0.875rem; transition: background 0.15s;
	}
	.tabs button.active { background: #1e1e35; color: #e0e0e0; }

	.form { display: flex; flex-direction: column; gap: 0.875rem; }

	label { display: flex; flex-direction: column; gap: 0.35rem; font-size: 0.8rem; color: #aaa; }
	input {
		padding: 0.55rem 0.75rem; border-radius: 7px; border: 1px solid #2a2a45;
		background: #0f0f1a; color: #e0e0e0; font-size: 0.9rem; outline: none;
	}
	input:focus { border-color: #4f46e5; }

	.error { color: #f87171; font-size: 0.8rem; }

	.submit {
		padding: 0.6rem; border-radius: 7px; border: none;
		background: #4f46e5; color: #fff; font-size: 0.9rem; cursor: pointer;
		margin-top: 0.25rem; transition: background 0.15s;
	}
	.submit:hover:not(:disabled) { background: #4338ca; }
	.submit:disabled { opacity: 0.5; cursor: not-allowed; }

	.divider {
		display: flex; align-items: center; gap: 0.75rem;
		margin: 1.25rem 0; color: #444; font-size: 0.75rem;
	}
	.divider::before, .divider::after { content: ''; flex: 1; height: 1px; background: #2a2a45; }

	.oauth { display: flex; flex-direction: column; gap: 0.625rem; }
	.oauth-btn {
		display: block; text-align: center; padding: 0.55rem;
		border-radius: 7px; border: 1px solid #2a2a45; font-size: 0.875rem;
		background: transparent; color: #ccc; transition: border-color 0.15s, color 0.15s;
	}
	.oauth-btn:hover { border-color: #555; color: #fff; }
</style>
