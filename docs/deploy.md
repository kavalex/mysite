# Deploy

This template ships as a single static binary. Deploy should not require Docker,
Node.js, npm, or a C compiler on the host — only `mysite` and a `data.db` file.

## TL;DR

```bash
make build                          # → dist/mysite
scp dist/mysite server:/usr/local/bin/mysite
ssh server 'systemctl restart mysite'
```

## Host requirements

- Linux/amd64 or arm64 (or macOS for staging), bare-metal or VM.
- Port 8080 (or any via `ADDR=:80`).
- A `data.db` file on persistent storage (`/var/lib/mysite/data.db`).
- systemd (or runit, OpenRC — same idea).

**Not required:** Docker, Node.js, a C compiler, systemd-resolved, a separate DB node.

## Build

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/mysite .
```

Flags:
- `CGO_ENABLED=0` — no musl/glibc dependency; the binary runs anywhere.
- `-trimpath` — strips local build paths from the binary (important for reproducibility).
- `-ldflags="-s -w"` — strips the symbol table and DWARF; smaller binary, no debug.

Size: ~13 MB.

## Cross-compilation

```bash
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/mysite-linux-amd64   .
GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/mysite-linux-arm64   .
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/mysite-darwin-arm64 .
```

Or `make build-all`.

## Production configuration

`.env` is replaced by a systemd `EnvironmentFile`, Vault, or Doppler.
File `mysite.env` (only `KEY=VALUE`, no secrets):

```ini
ENV=prod
ADDR=:8080
DATABASE_URL=/var/lib/mysite/data.db
BASE_URL=https://example.com
LOGIN_RATE_PER_MIN=30
LOGIN_BURST=5
TRUSTED_PROXIES=127.0.0.1/32,::1/128
TLS_CERT_FILE=/etc/letsencrypt/live/example.com/fullchain.pem
TLS_KEY_FILE=/etc/letsencrypt/live/example.com/privkey.pem
```

## systemd unit

See also the Caddyfile mentioned below. This unit is a hardening baseline.

```ini
[Unit]
Description=MySite
After=network.target

[Service]
Type=simple
User=mysite
WorkingDirectory=/var/lib/mysite
ExecStart=/usr/local/bin/mysite
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=5
EnvironmentFile=/etc/mysite/mysite.env

NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/mysite
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
RestrictNamespaces=true
RestrictRealtime=true
LockPersonality=true
MemoryDenyWriteExecute=true

[Install]
WantedBy=multi-user.target
```

`ExecReload` sends SIGHUP — the app re-reads `ENV`, `LOGIN_*`.
Everything else (`ADDR`, `TLS`, `DATABASE_URL`, timeouts) requires a full restart
(`systemctl restart mysite`).

## SIGHUP reload: what is applied, what is not

Detailed in [reference.md](reference.md) (section "SIGHUP behavior"). Short version:

| Change | SIGHUP | Restart |
|---|:-:|:-:|
| `ENV` | ✓ | |
| `LOGIN_RATE_PER_MIN` | ✓ | |
| `LOGIN_BURST` | ✓ | |
| `ADDR` | ✗ (warn) | ✓ |
| `DATABASE_URL` | ✗ (warn) | ✓ |
| `TLS_*` | ✗ (warn) | ✓ |
| `BASE_URL` | ✗ (warn) | ✓ |
| `*_TIMEOUT` | ✗ (warn) | ✓ |
| `TRUSTED_PROXIES` | ✗ (warn) | ✓ |

After SIGHUP, the log shows:

```
SIGHUP: reloading config
SIGHUP: applied  field=LOGIN_RATE_PER_MIN
SIGHUP: change requires restart  field=ADDR
```

## Reverse proxy: Caddy

We put Caddy (or nginx) in front of the Go server for TLS and gzip.

`Caddyfile`:

```
example.com {
    reverse_proxy localhost:8080 {
        header_up X-Forwarded-For {remote_host}
    }
    encode gzip zstd
}
```

Production: `TRUSTED_PROXIES=127.0.0.1/32,::1/128`, `ENV=prod`,
`BASE_URL=https://example.com`.

For nginx: similar setup with `proxy_set_header X-Real-IP $remote_addr;`
and the trusted-proxy list configured in the app.

## SQLite backups

```bash
# Hot backup (without stopping the service):
sqlite3 /var/lib/mysite/data.db ".backup /var/backups/mysite-$(date +%F-%H%M).db"

# Restore:
systemctl stop mysite
cp /var/backups/mysite-<timestamp>.db /var/lib/mysite/data.db
systemctl start mysite
```

**Cron:**
```
0 3 * * * /usr/bin/sqlite3 /var/lib/mysite/data.db ".backup /var/backups/mysite-$(date +\%F).db"
```

## Monitoring

- **/healthz** — liveness. Always `200` while the process is alive. With systemd `Type=simple` + `Restart=on-failure` this is enough.
- **/readyz** — readiness. `503` if the DB does not ping; `200` otherwise. For k8s readinessProbe or HAProxy `option httpchk`.
- **Logs (stdout)** — go to journald, or JSON output for Loki/Elasticsearch.

Structured JSON logs with `ENV=prod`:

```bash
journalctl -u mysite -o cat | jq -r '.msg'                    # readable messages
journalctl -u mysite -o cat | jq 'select(.status>=500)'     # only errors
```

## Secrets

- TLS keys in `/etc/letsencrypt/live/...`, mode `0600`, owner `mysite`.
- DB passwords — not needed for SQLite; if you connect an external DB (Postgres) with TLS, put the DSN in `EnvironmentFile` with mode `0640`.
- DO NOT put secrets in `.env` in the repo (only `.env.example` exists).

## When to migrate from SQLite to Postgres

SQLite here is for dev and small-scale production simplicity (~10^5 requests/day).
If you get:

- traffic > 100 RPS sustained;
- multiple writer replicas;
- large tables (> 100 GB);
- JOINs on 5+ tables in a hot path;

→ switch to Postgres. See [recipes/swap-db.md](recipes/swap-db.md) (placeholder;
a real recipe will appear the first time someone does this migration).

## Upgrade

1. Stop: `systemctl stop mysite`.
2. Drop in the new binary: `cp dist/mysite /usr/local/bin/mysite`.
3. Start: `systemctl start mysite`.
4. Verify: `curl -i http://localhost:8080/healthz`.

DB migrations are applied automatically on startup (`store.Migrate` in `main.go`).
Old migrations are not re-applied; new ones run in alphabetical order.
For rollbacks you write them manually; the template has no down-migrations.
Change the schema with additive steps only (new table / new column with default).
