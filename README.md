# Wackorder Discord Bot

A Star Citizen order management Discord bot. Players register component orders; any user can fulfill them. Orders flow through a simple, non-linear lifecycle tracked in SQLite.

## Order lifecycle

```
ordered ──→ ready ──→ done
        │         │
        └─────────┘
        (canceled from any non-done state, by creator only)
```

## Commands

| Command | Description |
|---------|-------------|
| `/order` | Open modal to place an order |
| `/order component:x quality:y quantity:n` | Place an order directly |
| `/order-list` | All pending orders (default) |
| `/order-list mode:self` | Your own orders (all statuses) |
| `/order-list mode:pending` | All unfinished orders (ordered / ready) |
| `/order-list mode:all` | Every order |
| `/order-list mode:booked` | Ready orders you last updated |
| `/order-list component:<name>` | Filter by component name (case-insensitive, combinable with mode) |
| `/order-list older-than:<d>` | Filter orders older than a duration (e.g. `7`, `2w`, `1mo`) |
| `/order-update id:<n>` | Show a status picker for an order |
| `/order-update id:<n> status:<s>` | Update an order directly |
| `/order-update my-book:done` | Mark all your booked (ready) orders as done |
| `/order-update my-book:ordered` | Return all your booked orders to ordered |
| `/order-cancel id:<n>` | Cancel one of your own orders |

**Status values:** `ready` · `done`

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_TOKEN` | Yes | — | Discord bot token |
| `CORP_ID` | Yes | — | Discord Corp (server) ID |
| `DB_PATH` | No | `wackorder.db` | SQLite database file path |
| `LOG_LEVEL` | No | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | No | `text` or `json` | Log format |

## Run natively

```bash
go build -o wackorder ./cmd/wackorder
DISCORD_TOKEN=your_token CORP_ID=your_server_id ./wackorder
```

## Run with Docker

```bash
docker build -t wackorder .
docker run -d \
  --name wackorder \
  -e DISCORD_TOKEN=your_token \
  -e CORP_ID=your_server_id \
  -v wackorder-data:/data \
  wackorder
```

The SQLite database is stored at `/data/wackorder.db` in the container. Mount a named volume to persist data across restarts.

## Development

```bash
go test ./...
go vet ./...
```

## Architecture

```
cmd/wackorder/        # entrypoint
internal/
  config/             # env-var loading
  db/                 # SQLite open + schema migration
  model/              # Order struct, Status enum, transition validation
  store/              # store.Repository interface + SQLite implementation
  bot/                # Discord session, slash commands, interaction handlers
```

The `store.Repository` interface decouples the bot from SQLite — swap in any backend by implementing the 11-method interface.
