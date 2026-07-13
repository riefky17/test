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
frontend/            3 independent Vite + React + TypeScript apps, one per service --
                      see "3 separate frontend apps" below
deploy/              HAProxy config, Cloudflare Tunnel config, systemd units, Postgres tuning
scripts/             build.sh (builds everything), backup.sh (nightly pg_dump -> R2/B2)
```

## 3 separate frontend apps, one shared login

Each app is its own independently-built Vite project, not a route inside a
shared bundle:

```
frontend/
  shared/               plain TypeScript, no build step of its own --
                          design-system/  ClayCard/ClayButton/ClayInput + clay.css
                          lib/            api client, auth context, IDR formatting,
                                          cross-app link helper (siblingApps.ts)
                          AppFrame.tsx    shared nav chrome
                          LoginPage.tsx   shared login screen
  apps/
    finance/            @family-app-suite/finance-app  (dev port 5173 -> :3001)
    geochat/            @family-app-suite/geochat-app   (dev port 5174 -> :3002)
    fitness/            @family-app-suite/fitness-app   (dev port 5175 -> :3003)
```

Each app imports `shared/` via a `@shared` path alias/Vite alias (plain
relative source, no internal npm package to publish or version). Every app
builds and deploys on its own — Finance Tracker's dist/ never contains a
byte of Geochat's code, and vice versa.

**Same login, no shared code at runtime:** the 3 apps authenticate against
the same `auth.users` table and set the same session cookie (scoped to
`COOKIE_DOMAIN`, e.g. `.yourdomain.com`). Log in on Finance Tracker and
Geochat/Fitness Tracker already see you as signed in — there's no SSO
service or token-passing involved, just one cookie the browser sends to
whichever subdomain a request goes to. The nav bar's links to the other 2
apps (`shared/AppFrame.tsx`) are plain cross-origin `<a>` tags, not
client-side routes, since each app really is a separate origin in
production.

## Why 3 separate services instead of one monolith

Splitting Finance/Geochat/Fitness into 3 Go binaries means a crash or bad
deploy in one (e.g. Finance) can never take down another's process (e.g.
Geochat's websocket, which carries SOS). Each runs under its own systemd
unit with `Restart=on-failure`. HAProxy routes by hostname to whichever
backend owns that hostname; see `deploy/haproxy/haproxy.cfg`.

All 3 share one Postgres instance with separate schemas (not separate
databases) so backups, tuning, and connection limits stay in one place
while each app's tables stay isolated.

## Geochat: a self-contained web messenger, no third-party bot

By design, Geochat does not route through Telegram, WhatsApp, or any other
outside service — it's its own web messenger, and location sharing runs on
the browser's native Geolocation permission (`frontend/apps/geochat/src/GeochatScreen.tsx`):
the user gets the browser's own "Allow location access?" prompt, nothing else
asks for consent, and no location data leaves this app.

SOS does two things on trigger (see `backend/geochat-svc/sos.go`):

1. Broadcasts over the open websocket to whichever family members are connected —
   this is the **only** delivery path there is, since there's no third-party fallback.
2. Logs the trigger twice, independently: a Postgres row (`geochat.sos_events`,
   including `recipients_online` — how many other family members were actually
   connected at that moment) and an append-only file (`SOS_LOG_PATH`, default
   `/var/log/geochat/sos.log`).

**Trade-off worth naming honestly:** because there's no fallback channel outside
this VPS, SOS delivery depends entirely on geochat-svc (and the VPS it runs on)
being up and the recipient's browser tab having an open websocket connection. If
that matters more than staying third-party-free, the natural next step is Web
Push (a service worker + the Push API) so a trigger can reach a phone even with
the tab closed — still no outside bot, but it does mean geochat-svc would need
to hold VAPID keys and each client's push subscription.

## Local development

Requires Go 1.22+, Node 18+, and a local Postgres.

```bash
# 1. create a database and apply migrations
createdb familyapps
for f in db/migrations/*.sql; do psql familyapps -f "$f"; done

# 2. seed the 3 accounts
cd backend
DATABASE_URL=postgres://localhost/familyapps go run ./cmd/seed papa:choose-a-password mami:choose-a-password echa:choose-a-password

# 3. run all 3 backends, each on its own port (see deploy/env.example)
DATABASE_URL=postgres://localhost/familyapps LISTEN_ADDR=127.0.0.1:3001 COOKIE_SECURE=false go run ./finance-svc
DATABASE_URL=postgres://localhost/familyapps LISTEN_ADDR=127.0.0.1:3002 COOKIE_SECURE=false go run ./geochat-svc
DATABASE_URL=postgres://localhost/familyapps LISTEN_ADDR=127.0.0.1:3003 COOKIE_SECURE=false go run ./fitness-svc

# 4. install once at the frontend root (npm workspaces), then run whichever app(s) you're working on
cd ../frontend
npm install
npm run dev --workspace=@family-app-suite/finance-app   # http://localhost:5173, proxies /api to :3001
npm run dev --workspace=@family-app-suite/geochat-app    # http://localhost:5174, proxies /api to :3002
npm run dev --workspace=@family-app-suite/fitness-app    # http://localhost:5175, proxies /api to :3003
```

Because cookies aren't port-scoped, logging in on `localhost:5173` also
authenticates `localhost:5174` and `:5175` in local dev — the same
mechanism `COOKIE_DOMAIN` provides across real subdomains in production.

## Building for deployment

```bash
./scripts/build.sh
```

Builds all 3 frontend apps and the 3 Go binaries, and lays out
`dist/<service>/{<service>,public/}` matching what each systemd unit in
`deploy/systemd/` expects at `/opt/family-app-suite/<service>/` — each
service gets only its own app's build in its `public/`.

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
