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
| `/order-view` | Your own orders (all statuses) |
| `/order-view view:self` | Your own orders (all statuses) |
| `/order-view view:pending` | All unfinished orders (ordered / ready) |
| `/order-view view:all` | Every order |
| `/order-view component:<name>` | Search by component name (case-insensitive) |
| `/order-view since:<date>` | Orders created since a date (YYYY-MM-DD or RFC3339) |
| `/order-update` | Open modal to update an order's status |
| `/order-update id:<n> status:<s>` | Update an order directly |
| `/order-cancel id:<n>` | Cancel one of your own orders |

**Status values:** `ready` · `done`

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_TOKEN` | Yes | — | Discord bot token |
| `CORP_ID` | Yes | — | Discord guild (server) ID |
| `DB_PATH` | No | `wackorder.db` | SQLite database file path |
| `LOG_LEVEL` | No | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | No | `text` or `json` | Log format |

## Run natively

```bash
go build -o wackorder ./cmd/wackorder
DISCORD_TOKEN=your_token CORP_ID=your_guild_id ./wackorder
```

## Run with Docker

```bash
docker build -t wackorder .
docker run -d \
  --name wackorder \
  -e DISCORD_TOKEN=your_token \
  -e CORP_ID=your_guild_id \
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

The `store.Repository` interface decouples the bot from SQLite — swap in any backend by implementing the 8-method interface.
