# advanced-systembolaget-system

Full-stack web application for searching, syncing, and collaborating on Systembolaget's product catalog. Includes multi-user auth, shared lists, tasting events with scoring, a roll game, public shareable lists, and a standalone CLI for batch exports.

Built with Go (Gin), Vue 3 (PrimeVue), and SQLite.

## Quick start

### Docker (recommended)

Create a `config.json`:

```json
{
  "api_key": "",
  "admin_user": "admin",
  "admin_pass": "changeme"
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

The server reads `config.json` for admin credentials and API key, creates a SQLite database in `data/`, and serves the frontend on port 8080 (override with `$PORT`).

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

Organize tastings: create an event, invite users, add beers, and score each product 0-10. Supports importing beers directly from a shared list. Lock events when scoring is complete.

### Roll game

A game mode for events: add products to a pool, roll to randomly draw one, then accept or veto the pick. Each user gets one veto. Admins can undo vetoes/consumed items and reset the game.

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
| Events | CRUD on `/events`, attendees, beers, scores, list import |
| Roll game | `/events/:id/roll` (state, roll, accept, veto, reset) |
| Shared lists | CRUD on `/shared-lists`, items, locking, sharing, public view at `/public/shared-list/:uuid` |
| Admin | `/admin/users` CRUD, `POST /admin/impersonate/:id`, `DELETE /admin/comments/:id`, `DELETE /admin/products` |

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

`config.json` in the working directory:

```json
{
  "api_key": "",
  "admin_user": "admin",
  "admin_pass": "changeme"
}
```

| Field | Description |
|-------|-------------|
| `api_key` | Systembolaget API key (auto-fetched via `--get-key` or the web UI) |
| `admin_user` | Initial admin username (seeded on first run) |
| `admin_pass` | Initial admin password (hashed with bcrypt on seed) |
