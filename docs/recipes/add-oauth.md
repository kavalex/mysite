# Recipe: add OAuth (Google / GitHub)

**Status:** placeholder.

OAuth adds at least **1 dependency** (`golang.org/x/oauth2`; the rest can be
written on top of `net/http`).

Points of change:

1. **`internal/auth/oauth.go`** (new) — `AuthCodeURL`, `Exchange(code)`.
2. **`internal/store/users.go`** — add `oauth_provider` + `oauth_subject` columns.
3. **`web/templates/pages/login.html`** — add "Sign in with Google" buttons.
4. **`internal/server/server.go`** — routes `/oauth/{provider}/start`, `/oauth/{provider}/callback`.
5. **`internal/auth/session.go`** — no change; the session stays cookie-based.

CBOR / JWE / HMAC for the state token — minimal PKCE flow. No
`gorilla/sessions` / `oauth2/v2` — stdlib + `oauth2` is enough.

The recipe will land with the first PR.
