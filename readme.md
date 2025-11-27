# Air Monitoring API

Small Go/Chi service for storing and streaming sensor measurements over HTTP and Server-Sent Events (SSE). Data is persisted in SQLite with simple pagination and runtime-tunable settings.

## Quick Start
- Build locally: `make build` (writes `bin/air-server`).
- Run in background with defaults: `make start` (port `4001`, env `development`, DB `api.db`). Tail logs with `make logs`; stop with `make stop`.
- Run directly: `./bin/air-server -port=4001 -env=development -db=api.db`.
- Clean artifacts: `make clean`. Cross-compile static Linux binary on macOS: `make linux_release_on_mac` (requires Docker).

## API Surface
- Health: `GET /health` returns 200.
- Slow test: `GET /slow` or `/slow/{seconds}` to simulate latency.
- Measurements:
  - `GET /api/measurements?limit=50&cursor=...` returns `{items, next_cursor, has_more}` ordered by `created_at`.
  - `POST /api/measurements` to ingest measurements.
  - `GET /api/measurements/stream` opens SSE feed (`event: measurements`) pushing created measurements.
- Settings:
  - `GET /api/settings` lists keys; `GET /api/settings/{key}` fetches one (falls back to defaults).
  - `POST /api/settings/{key}` updates a value; keys include `store_interval` (seconds between accepted writes) and `max_age` (seconds to retain).

### Example Requests
```bash
# Create two measurements (timestamp optional; defaults to now)
curl -X POST http://localhost:4001/api/measurements \
  -H "Content-Type: application/json" \
  -d '{"values":[{"sensor":"sensor-1","parameter":"pm10","value":12.3,"unit":"µg/m3"},{"sensor":"sensor-1","value":55,"unit":"%"}]}'

# Paginate
curl "http://localhost:4001/api/measurements?limit=10&cursor=${CURSOR}"

# Stream new measurements
curl -N http://localhost:4001/api/measurements/stream
```

## Development Notes
- Go version: `go 1.23.3` (see `go.mod`). Dependencies managed via `go mod tidy`.
- Code lives in `cmd/api`: handlers in `handler/`, storage in `storage/`, models in `models/`, settings cache in `settings/`, pagination helpers in `pagination/`.
- SQLite schema is created automatically on startup; defaults are populated from `settings.DefaultSettings`.
- Tests: none yet; add `_test.go` files and run `go test ./...`.
