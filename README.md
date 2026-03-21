![Go](https://img.shields.io/badge/Go-1.26.1-00ADD8?logo=go&logoColor=white)
![BoltDB](https://img.shields.io/badge/bbolt-1.4.3-4C8CBF?logo=sqlite&logoColor=white)

# Simple Bot 🤖

Simple Bot is a modern Go application for analyzing and automating item management in a live market in massive multiplayer online game (MMO).

## Features ✨
- Analyze and compare market items and their values
- Automatic buying and selling of items
- Store and retrieve results in a fast local database
- Bubble Tea terminal UI with market scan and DB operations
- Capture and persist item `value` data

## Requirements 📦
- Go 1.18+
- [bbolt](https://github.com/etcd-io/bbolt) for local storage

## Getting Started 🚀
1. Clone the repository
2. Copy `.env.template` to `.env` and set your environment variables
3. Build and run the application:
   ```sh
   go run ./cmd/simple-bot/main.go
   ```

## Run Modes
- `inspect <start_id> <end_id>`: bulk inspect IDs and persist values
- `analyze`: market analysis in terminal logs
- `ui`: interactive TUI (scan + local DB management)

## Local DB UX
- `update range` progress shows completion percentage (`%`) and failures while processing IDs.

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

## Architecture Diagram
```mermaid
flowchart TB
      CLI[cmd/simple-bot/main.go] --> CFG[configs/config.go\nLoad .env and APP_BASE_URL]
      CLI --> HTTP[internal/utils/http.go\nHTTP client from call.txt]
      CLI --> DB[internal/database/keystore.go\nbbolt Store]

      CLI --> MODE{Execution mode}
      MODE --> INSPECT[inspect]
      MODE --> ANALYZE[analyze]
      MODE --> UI[ui]

      INSPECT --> AIP[utils.AnalyzeInspectParallel]
      ANALYZE --> AM[utils.AnalyzeMarket]
      UI --> APP[internal/ui/app.go\nBubble Tea Model]
      APP --> SM[utils.ScanMarket]
      APP --> REFRESH[utils.RefreshItemValue]

      AIP --> STATS[(POST /api/item/stats-v2/:id)]
      AM --> LIST[(GET /market/listings)]
      SM --> LIST
      REFRESH --> STATS

      STATS --> PARSE[utils.inspectItemStats\nvalue]
      PARSE --> DB
      LIST --> MP[utils.parseMarketPage]
      MP --> DB

      DB --> KV[(bucket: kv)]
```

## Internal Functional Flow (Modules)
```mermaid
flowchart TD
      START[UI mode: start scan] --> UISCAN[internal/ui/app.go\nstartScan -> ScanMarket]
      UISCAN --> SCAN[internal/utils/analyze.go\nscanLevelRange]
      SCAN --> FETCHLIST[HTTP GET /market/listings]
      FETCHLIST --> PARSEPAGE[parseMarketPage]

      PARSEPAGE --> FORITEM{For each parsed item}
      FORITEM --> GETVALUE[Store.Get item value]
      GETVALUE --> BUILD[Build MarketItem\nGold + Value]

      BUILD --> VIEW[UI renderRow\nshow Cost, Value, Profit]
      VIEW --> DECIDE{Profitable?}
      DECIDE -->|yes| BUY[BuyItem POST /api/market/buy/:id]
      DECIDE -->|no| NEXT[Next item/page]
      BUY --> NEXT
```