# Wackorder Discord Bot

A Star Citizen order management Discord bot. Players register component orders; any user can fulfill them. Orders flow through a simple lifecycle tracked in SQLite.

## Order lifecycle

```
ordered ──→ ready ──→ done
        ↑     │
        └─────┘
```

- Any user can move an order from `ordered` → `ready`, `ready` → `done`, or `ready` → `ordered`.
- `ordered` → `done` is also allowed (skip the ready step).
- `done` and `canceled` are terminal states — no further transitions.
- Only the order creator (or a user with an admin role) can cancel an order with `/order-cancel`.

## Commands

### Placing orders

| Command | Description |
|---------|-------------|
| `/order` | Open a full modal (component + quality + quantity) |
| `/order component:x` | Open a partial modal (quality + quantity pre-filled) |
| `/order component:x quantity:n` | Place an order directly (quality defaults to 0) |
| `/order component:x quality:y quantity:n` | Place an order directly |

The `component` field uses autocomplete and must match one of the supported resources (see `/order-help` for the full list). Quality is an integer (e.g. `750`); leaving it blank or `0` means no quality requirement. Quantity is in cSCU or Units depending on the resource type.

### Listing orders

All `/order-list` responses are ephemeral (visible only to you).

| Command | Description |
|---------|-------------|
| `/order-list` | Pending orders — `ordered` and `ready` (default) |
| `/order-list mode:pending` | Same as default |
| `/order-list mode:self` | Your own orders (all statuses) |
| `/order-list mode:all` | Every order regardless of status |
| `/order-list mode:booked` | Orders with status `ready` that you last updated |
| `/order-list mode:done` | All completed orders (any creator) |
| `/order-list component:<name>` | Filter by component name (case-insensitive substring, combinable with `mode`) |
| `/order-list older-than:<d>` | Filter orders created before the given age (combinable with `mode` and `component`) |

`older-than` accepts: plain integer days (`7`), days (`7d`), weeks (`2w`), months (`1mo`, = 30 days).

### Updating orders

| Command | Description |
|---------|-------------|
| `/order-update id:<n>` | Show an interactive status picker for order `n` |
| `/order-update id:<n> status:<s>` | Update order `n` directly to status `s` |
| `/order-update booked:done` | Mark all your booked (`ready`) orders as done |
| `/order-update booked:ordered` | Return all your booked orders to `ordered` |

**Valid status values:** `ready` · `done` · `ordered`

### Other commands

| Command | Description |
|---------|-------------|
| `/order-cancel id:<n>` | Cancel order `n` (creator or admin only) |
| `/order-detail id:<n>` | Show full details of a single order (ephemeral) |
| `/order-stats` | Summary of pending quantities per resource with a day-by-day histogram |
| `/order-stats jours:<1-32>` | Same with a custom histogram width (default 32 days) |
| `/order-help` | Show the workflow guide and the full list of supported resources (ephemeral) |

`/order-stats` renders interactive per-resource filter buttons (up to 25) that open a filtered order list on click.

## Automatic cleanup

`done` orders are automatically deleted after **14 days**. The pruner runs at startup and then daily at midnight UTC.

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_TOKEN` | Yes | — | Discord bot token |
| `CORP_ID` | Yes | — | Discord guild (server) ID to register slash commands on |
| `DB_PATH` | No | `wackorder.db` | SQLite database file path |
| `LOG_LEVEL` | No | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | No | `text` | `text` or `json` |
| `LOG_CHANNEL_ID` | No | — | Discord channel ID for action audit logs; disabled if empty |
| `RECAP_CHANNEL_ID` | No | — | Discord channel ID for daily recap; disabled if empty |
| `ADMIN_ROLE_IDS` | No | — | Comma-separated Discord role IDs whose members can cancel any order |

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

The SQLite database is stored at `/data/wackorder.db` in the container (set via `ENV DB_PATH` in the Dockerfile). Mount a named volume to persist data across restarts.

The runtime image is `gcr.io/distroless/static-debian12:nonroot` — no shell, no package manager.

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
  db/                 # SQLite open + WAL/FK pragmas + schema init
  model/              # Order struct, Status enum, transition validation
  store/              # store.Repository interface + SQLite implementation
  bot/                # Discord session, slash commands, interaction handlers, daily pruner
```

The `store.Repository` interface decouples the bot from SQLite — swap in any backend by implementing the 12-method interface (`Create`, `GetByID`, `ListByCreator`, `ListPending`, `ListAll`, `SearchByComponent`, `ListSince`, `ListBefore`, `UpdateStatus`, `ListReadyByUpdater`, `ListDone`, `Prune`).
