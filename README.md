# Family App Suite

Three apps for a family of 3 (Papa, Mami, Echa), built to run on a single
1 vCPU / 1GB RAM VPS:

- **Finance Tracker** — IDR income/expense tracking
- **Geochat** — family chat, location sharing, and an SOS button
- **Fitness Tracker** — home gym workout plans and session logging

Architecture, sizing rationale, and every tradeoff below are documented in
detail in the original tech-stack doc this repo implements.

## Layout

```
backend/            Go module: 3 services + shared internal packages
  internal/auth/     shared login/session logic (bcrypt + cookie sessions)
  internal/config/   env var loading
  internal/db/       shared Postgres pool
  internal/spa/      serves the built frontend from each service
  finance-svc/       Finance Tracker backend (127.0.0.1:3001)
  geochat-svc/        Geochat backend incl. websocket + SOS (127.0.0.1:3002)
  fitness-svc/       Fitness Tracker backend (127.0.0.1:3003)
  cmd/seed/          one-off script to create the 3 family accounts
db/migrations/       SQL migrations, schema-separated per app (auth/finance/geochat/fitness)
frontend/            Vite + React + TypeScript SPA, shared claymorphism design system,
                      one route tree per app behind a single login
deploy/              HAProxy config, Cloudflare Tunnel config, systemd units, Postgres tuning
scripts/             build.sh (builds everything), backup.sh (nightly pg_dump -> R2/B2)
```

## Why 3 separate services instead of one monolith

Splitting Finance/Geochat/Fitness into 3 Go binaries means a crash or bad
deploy in one (e.g. Finance) can never take down another's process (e.g.
Geochat's websocket, which carries SOS). Each runs under its own systemd
unit with `Restart=on-failure`. HAProxy routes by hostname to whichever
backend owns that hostname; see `deploy/haproxy/haproxy.cfg`.

All 3 share one Postgres instance with separate schemas (not separate
databases) so backups, tuning, and connection limits stay in one place
while each app's tables stay isolated.

## SOS: the one non-negotiable part of this stack

Geochat's SOS button does three things on trigger (see
`backend/geochat-svc/sos.go`):

1. Broadcasts over the open websocket to whichever family members are connected.
2. Fires a Telegram Bot API `sendMessage` + `sendLocation` to the other two
   phones — a fallback channel that does **not** depend on this VPS being up.
3. Logs the trigger twice, independently: a Postgres row (`geochat.sos_events`)
   and an append-only file (`SOS_LOG_PATH`, default `/var/log/geochat/sos.log`).

**Before trusting this in production, test the Telegram fallback with the
VPS's geochat-svc process stopped**, not just in normal operation.

## Local development

Requires Go 1.22+, Node 18+, and a local Postgres.

```bash
# 1. create a database and apply migrations
createdb familyapps
for f in db/migrations/*.sql; do psql familyapps -f "$f"; done

# 2. seed the 3 accounts
cd backend
DATABASE_URL=postgres://localhost/familyapps go run ./cmd/seed papa:choose-a-password mami:choose-a-password echa:choose-a-password

# 3. run a service (repeat per service on its own port; see deploy/env.example)
DATABASE_URL=postgres://localhost/familyapps LISTEN_ADDR=127.0.0.1:3001 COOKIE_SECURE=false go run ./finance-svc

# 4. run the frontend (proxies /api/* to the 3 ports above -- see frontend/vite.config.ts)
cd ../frontend
npm install
npm run dev
```

## Building for deployment

```bash
./scripts/build.sh
```

Builds the frontend once and the 3 Go binaries, and lays out
`dist/<service>/{<service>,public/}` matching what each systemd unit in
`deploy/systemd/` expects at `/opt/family-app-suite/<service>/`.

## Deploying

See `deploy/`:

- `haproxy/haproxy.cfg` — hostname-based routing to the 3 backends, stats page on loopback
- `cloudflared/config.yml` — tunnel ingress rules (no inbound ports opened on the VPS)
- `systemd/*.service` — one unit per Go service, all `Restart=on-failure`
- `postgresql.tuning.conf` — sizing for a 1GB box, not defaults meant for multi-GB machines
- `env.example` — the env vars each service reads (copy to `/etc/family-app-suite/<service>.env`)

Nightly backups: `scripts/backup.sh` (`pg_dump` -> R2/B2 via `rclone`, 14-day retention by default).

## Fitness content note

The seeded exercises and plan "styles" in `db/migrations/004_fitness.sql`
and the Fitness Tracker UI follow public, generic training principles
(progressive overload, push/pull/legs and upper/lower splits,
posterior-chain-focused hypertrophy work) — they are not a transcription
of any specific paid program's exact structure.
