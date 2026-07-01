# ADR 005: Auth -- JWT + Rotating Refresh Token over Server-Side Sessions

**Status:** Accepted   ·   **Date:** 2026-06-27

## Context

The app requires authentication for all data endpoints.
Auth methods: email/password, Google OAuth, Discord OAuth.
The frontend is a SPA running in the browser -- it cannot securely store a long-lived secret in JavaScript memory or localStorage.
The backend is a stateless Go API; minimizing server-side session state is preferred for simplicity.

## Decision

Issue a short-lived JWT access token (15-minute expiry, signed with `AUTH_SECRET` env var, contains `{sub: userId}`) plus a long-lived refresh token (30-day expiry, 32 random bytes, bcrypt-hashed and stored in `refresh_tokens` table, sent as `HttpOnly; Secure; SameSite=Strict` cookie).
Rotate the refresh token on every use: old row deleted, new row inserted.
Access token stored in a Svelte writable store (memory only -- lost on page reload).
A fetch wrapper catches 401 responses, calls `POST /auth/refresh`, and retries the original request transparently.

## Options Considered

### Option A -- JWT access token + rotating refresh token (HttpOnly cookie)
- Pros: access token is stateless -- the API verifies it by signature alone, no DB lookup per request; short TTL (15 min) limits the blast radius of a stolen access token; refresh token is HttpOnly, so JavaScript cannot read it (XSS cannot steal it); rotation means a stolen refresh token can be used at most once before being invalidated; scales horizontally without shared session storage.
- Cons: refresh token requires a DB table (`refresh_tokens`), so the system is not fully stateless; the 401-intercept-and-retry pattern adds client-side complexity; access token cannot be revoked mid-TTL (a logged-out user's 15-minute token remains valid until expiry).

### Option B -- Server-side sessions (cookie + session store)
- Pros: fully revocable at any time; simpler client-side code (browser sends session cookie automatically); no token management logic.
- Cons: requires shared session storage (Redis or DB) accessible by all API instances -- adds infrastructure; every authenticated request hits the session store; harder to scale horizontally; the developer is less familiar with this pattern in Go.

### Option C -- Long-lived JWT (no refresh token)
- Pros: simplest implementation -- no refresh flow, no DB table.
- Cons: a stolen token is valid until expiry with no revocation path; long-lived JWTs in localStorage are an XSS target; rejected on security grounds.

## Consequences

- **Positive:** stateless access token verification keeps hot-path latency low (no DB lookup per request). XSS cannot steal the refresh token. Token rotation limits the window for a stolen cookie.
- **Negative:** the `refresh_tokens` table must be cleaned up periodically (expired rows accumulate). The 15-minute access token window means a compromised token has a brief but non-zero validity period. The 401-retry pattern must be carefully implemented to avoid retry loops on legitimately rejected requests.

## Security implications

- `AUTH_SECRET` must be a high-entropy random string stored in an env var -- never committed to the repo.
- Refresh token is stored as a bcrypt hash -- the raw token travels only in the HttpOnly cookie. A DB breach does not expose raw tokens.
- `SameSite=Strict` on the refresh cookie prevents CSRF on the `/auth/refresh` endpoint.
- OAuth `state` parameter validated on callback to prevent CSRF on the OAuth flows.
- The `/auth/refresh` endpoint must delete the old refresh token row and issue a new one atomically (within a DB transaction) to prevent race conditions on concurrent refresh calls.

## Open question

Refresh token TTL: fixed 30-day expiry vs. sliding window (extend on each use).
Ship fixed TTL first. Revisit if users report being logged out unexpectedly after inactivity.

## Related Docs

- Tech spec section 3.7 (auth flow): `docs/tech-spec.md`
- Tech spec section 5 (security): `docs/tech-spec.md`
