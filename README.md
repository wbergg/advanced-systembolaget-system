# advanced-systembolaget-system

Full-stack web application for searching, syncing, and collaborating on Systembolaget's product catalog. Includes multi-user auth, shared lists, tasting events with scoring, a roll game, public shareable lists, and a standalone CLI for batch exports.

Built with Go (Gin), Vue 3 (PrimeVue), and SQLite.

## Quick start

### Docker (recommended)

Create a `config.json` (see `config_example.json`):

```json
{
  "api_key": "",
  "admin_user": "admin",
  "admin_pass": "changeme",
  "listen_ip": "0.0.0.0",
  "port": "8080",
  "printer": {
    "enabled": false,
    "url": ""
  }
}
```

```bash
docker compose up -d
```

The app is available at `http://localhost:8080`. Log in with the admin credentials from your config, then use the Sync panel to fetch products from Systembolaget.

### Build from source

```bash
# Build frontend
cd frontend && npm ci && npm run build && cd ..

# Build API server
go build -o systemet-ass ./cmd/api

# Build CLI tool
go build -o systemet-poll-cli ./cmd/cli
```

Run the server:

```bash
./systemet-ass
```

The server reads `config.json` for admin credentials, API key, and listen settings. It creates a SQLite database in `data/` and serves the frontend.

## Architecture

```
cmd/
  api/          API server entry point
  cli/          Standalone CLI for batch product fetching
internal/
  auth/         JWT authentication & middleware
  db/           SQLite schema, queries (products, events, shared lists, users, etc.)
  handlers/     HTTP handlers (login, user management)
  systembolaget/ Systembolaget API client, key extraction, search params
frontend/       Vue 3 + TypeScript SPA (PrimeVue, Pinia)
```

The frontend is embedded into the Go binary via `go:embed` and served as a single-page app with fallback routing.

## Features

### Product search & sync

Search and filter the full Systembolaget catalog. Sync products from Systembolaget's API directly from the web UI (with real-time progress via SSE). Filters include category, price range, ABV%, country, packaging, producer, vintage, and free text.

### Shared lists

Create collaborative shopping lists, add products with quantities, and share them with other users. Lock lists when finalized. Each list gets a public link (`/delad-lista/{uuid}`) viewable without login. Supports JSON export/import for transferring lists between instances.

### Tasting events

Organize tastings: create an event, invite users, add beers, and score each product 0-10. Beers can be imported from a shared list or added manually. Lock events when scoring is complete, then archive past events to keep the active list clean (admins can browse the archive separately).

### Roll game

A game mode for events: add products to a pool, roll to randomly draw one, then accept or veto the pick. Each user gets one veto, and each item can only be vetoed once. The host or an admin can undo vetoes, undo consumed items, replace the pool from a shared list, or reset the game.

Decision time (seconds between roll and accept/veto) is recorded per turn and exposed in the API. Optional receipt-printer integration prints a slip for each roll, accept, and veto (see `printer` config below).

Only one roll event can be public at a time. Publishing an event makes it available at `/roll` without authentication. If no event is published, the page shows "Currently no active event." Admins can additionally hide an event from the regular event list.

#### Public roll API

The public roll endpoints require no authentication, so external apps can integrate with the game:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/public/roll` | Get game state, participants, and pending turn |
| `POST` | `/api/public/roll` | Perform a roll (`{"userId": N}`) |
| `POST` | `/api/public/roll/:turnId/accept` | Accept the pending roll |
| `POST` | `/api/public/roll/:turnId/veto` | Veto the pending roll |

`GET /api/public/roll` returns the event name, participant list, pool/consumed/vetoed counts, and the current pending turn (including `canVeto`, country, ABV%, volume, and `decisionSeconds` once resolved). Returns `404` when no event is published.

### Comments & notes

Leave comments on products (visible to all users) and personal notes (visible only to you).

### User management

Admin panel for creating/managing users. Role-based access (admin/user), password requirements (10+ chars, uppercase, digit), audit logging, and admin impersonation for debugging.

## API

All endpoints are under `/api/`. Authentication uses JWT tokens (24h expiry).

| Area | Endpoints |
|------|-----------|
| Auth | `POST /login`, `GET /me`, `PUT /me/password` |
| Products | `GET /products`, `GET /products/:id`, `GET /products/distinct/:column`, `PATCH /products/:id/notes` |
| Comments | `GET /products/:id/comments`, `POST /products/:id/comments` |
| Sync | `POST /sync` (SSE), `POST /key/refresh`, `GET /key/status` |
| Events | CRUD on `/events`, attendees, beers, scores, list import, locking, `POST /events/:id/archive`, `POST /events/:id/unarchive` |
| Roll game | `/events/:id/roll` (state, roll, accept, veto, reset, undo veto/consumed), `/events/:id/public` (publish), `/events/:id/visibility` (hide), `/public/roll` (public access) |
| Shared lists | CRUD on `/shared-lists`, items, locking, sharing, public view at `/public/shared-list/:uuid` |
| Admin | `/admin/users` CRUD, `POST /admin/impersonate/:id`, `DELETE /admin/comments/:id`, `POST /admin/products`, `DELETE /admin/products/:id`, `DELETE /admin/products`, `GET /admin/events/archived`, `GET /admin/debug/sb-probe/:number` |

## CLI tool

The standalone CLI (`cmd/cli`) fetches products directly from Systembolaget's API and outputs JSON Lines.

```bash
# Fetch API key
./systemet-poll-cli --get-key

# Beers in cans under 15 SEK
./systemet-poll-cli -kategori Öl -forpackning Burk -pris-till 15

# Free text search, save to file
./systemet-poll-cli -q "imperial stout" -o stouts.jsonl
```

All filter flags use Swedish names matching systembolaget.se URL parameters (`-pris-fran`, `-pris-till`, `-alkoholhalt-fran`, `-kategori`, `-typ`, `-land`, `-forpackning`, `-volym-fran`, `-sortera-pa`, `-q`, etc.).

### Adding new CLI filters

Add one line to `paramMappings` in `internal/systembolaget/params.go`:

```go
{"sockerhalt-till", "sugarContentGramPer100ml.max", "Max sugar (g/100ml)"},
```

## Configuration

`config.json` in the working directory (see `config_example.json`):

```json
{
  "api_key": "",
  "admin_user": "admin",
  "admin_pass": "changeme",
  "listen_ip": "0.0.0.0",
  "port": "8080",
  "printer": {
    "enabled": false,
    "url": "http://printer.local/cgi-bin/print"
  }
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `api_key` | Systembolaget API key (auto-fetched via `--get-key` or the web UI) | |
| `admin_user` | Initial admin username (seeded on first run) | *required* |
| `admin_pass` | Initial admin password (hashed with bcrypt on seed) | *required* |
| `listen_ip` | IP address to bind to | `0.0.0.0` |
| `port` | Port to listen on | `8080` |
| `printer.enabled` | Enable receipt-printer integration for roll events | `false` |
| `printer.url` | HTTP endpoint that accepts `stext`/`print_text`/`cut_command` form posts | |

### Printer integration

When enabled, the server queues a print job for each roll, accept, and veto in the roll game. Jobs are POSTed as `application/x-www-form-urlencoded` to `printer.url`. Jobs are processed serially by a single worker, with a short delay before paper-cut commands so the print buffer can flush. If the printer is unreachable, jobs are logged and dropped — they don't block the API.
