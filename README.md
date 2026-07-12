![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8?logo=go&logoColor=white)
![BoltDB](https://img.shields.io/badge/bbolt-1.4.3-4C8CBF?logo=sqlite&logoColor=white)

# Simple Bot 🤖

Simple Bot is a Go application for analyzing and automating item management in a live market in a massive multiplayer online game (MMO). It scans market listings, compares item value against cost, auto-buys profitable deals, and persists inspected item values in a local bbolt database — all driven from a single binary with a Bubble Tea terminal UI.

![Simple Bot TUI menu](docs/screenshots/tui-menu.png)

## Requirements 📦
- Go 1.26+
- [bbolt](https://github.com/etcd-io/bbolt) for local storage

## Getting Started 🚀
1. Clone the repository
2. Copy `.env.template` to `.env` and set `APP_BASE_URL` (optionally `DB_PATH`, defaults to `internal/database/data.db`)
3. Create a `call.txt` file with the raw `curl` command (headers + cookie) used to authenticate against the market API — copy it from your browser's dev tools ("Copy as cURL")
4. Build and run the application:
   ```sh
   go run ./cmd/simple-bot ui
   ```

   Or use the provided Makefile targets:
   ```sh
   make go-build                    # builds dev/simple-bot
   make analyze-market               # runs analyze mode, logs to output.log
   make inspect-items INIT=1 END=100 # runs inspect mode for an ID range
   ```

## Run Modes
- `inspect <start_id> <end_id>`: bulk inspect IDs and persist values
- `analyze`: market analysis in terminal logs
- `ui`: interactive TUI (scan + local DB management)
- `version` (or `--version`, `-v`): prints current application version

## Internal Modules (Detailed)
- `cmd/simple-bot/main.go`
   - Entry point
   - Loads config, creates HTTP client, opens bbolt store, routes mode
- `configs/config.go`
   - Loads environment variables (mainly `APP_BASE_URL`)
- `internal/utils`
   - HTTP calls, parsing, inspect flow, market scan, buy operations
- `internal/ui`
   - Bubble Tea app state, scan view, DB view, range updates
- `internal/database/keystore.go`
   - bbolt storage abstraction for item values (`kv` bucket)
- `internal/models`
   - DTOs and domain models for market items and inspect payloads

## Internal Functional Flow (Modules)
```mermaid
flowchart TD
      START[Application start] --> CHOICE{Choose Mode}

      CHOICE --> INSPECT[inspect mode]
      CHOICE --> ANALYZE[analyze mode]
      CHOICE --> UI[ui mode]

   INSPECT --> STATS[stats-v2 API]
      STATS --> STORE1[save value in bbolt]

   ANALYZE --> LIST1[market listings API]
      LIST1 --> PARSE1[parse listings + read stored values]
      PARSE1 --> BUY1{profitable item}
      BUY1 -->|yes| BUYAPI[buy API]
      BUY1 -->|no| END1[continue analysis]
      BUYAPI --> END1

   UI --> UICHOICE{User action in UI}
      UICHOICE --> SCAN[Scan market]
      UICHOICE --> DBVIEW[Local DB view]

      SCAN --> LIST2[market listings API]
      LIST2 --> PARSE2[parse listings + render rows]

      DBVIEW --> DBACT{DB action}
      DBACT --> UPDATEONE[update selected]
      DBACT --> UPDATERANGE[update range]
      DBACT --> SEARCH[search / browse]

      UPDATEONE --> STATS2[stats-v2 API]
      STATS2 --> STORE2[update value in bbolt]
      UPDATERANGE --> STATS2
      SEARCH --> READDB[read from bbolt]
```