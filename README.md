
# Simple Bot 🤖

Simple Bot is a modern Go application for analyzing and automating item management in a live market in massive multiplayer online game (MMO).

## Features ✨
- Analyze and compare market items and their values
- Automatic buying and selling of items
- Store and retrieve results in a fast local database

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