# Recipe: add SSE / WebSocket

**Status:** placeholder.

For **Server-Sent Events** no external dependency is needed:

- The handler writes `Content-Type: text/event-stream` and calls `Flush()`
  after each line (`Flush` is already forwarded by `responseRecorder` in
  [`internal/middleware/logging.go`](../../internal/middleware/logging.go)).
- Stateful SSE: a new `*Runtime` field with `map[*client]struct{}` and a
  broadcast goroutine.
- CSP impact: SSE is long-poll, not WebSocket; CSP does not block it.

For **WebSocket upgrade** — at least 1 dep
(`nhooyr.io/websocket`, the modern successor to `gobwas/ws`). Points of change:

1. `internal/server/server.go` — special handling for the WebSocket upgrade;
   neither CSRF nor rate-limit through the usual middleware fits; you need a
   dedicated handler.
2. CSP — `connect-src 'self'` (or a specific origin); no WebSocket schemes.
3. Session auth — the cookie still validates; the `RequireLogin` middleware is
   used as usual.

SSE / WebSocket are not needed for the template today. Add them when the first
real feature requires them.
