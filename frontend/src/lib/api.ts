export const BASE = 'http://localhost:8080';

let _token: string | null = null;

export function setApiToken(t: string | null) { _token = t; }
export function getApiToken() { return _token; }

// Authenticated fetch wrapper. Intercepts 401, attempts silent refresh, retries once.
export async function api(path: string, init?: RequestInit): Promise<Response> {
	const makeHeaders = (): Record<string, string> => ({
		'Content-Type': 'application/json',
		...(init?.headers as Record<string, string>),
		...(_token ? { Authorization: `Bearer ${_token}` } : {}),
	});

	let res = await fetch(BASE + path, { ...init, credentials: 'include', headers: makeHeaders() });

	if (res.status === 401 && _token) {
		const ref = await fetch(`${BASE}/auth/refresh`, { method: 'POST', credentials: 'include' });
		if (ref.ok) {
			const data = await ref.json();
			_token = data.access_token;
			res = await fetch(BASE + path, { ...init, credentials: 'include', headers: makeHeaders() });
		} else {
			_token = null;
		}
	}

	return res;
}
