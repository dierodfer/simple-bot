#!make

.PHONY: clean

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

go-download: # Build the binary
	@go mod download

go-run: go-download # Run the binary
	@go run cmd/simple-bot/main.go > output.log

go-build: go-download # Build the binary
	@go build -o dev/simple-bot main.go

fill-baseurl:
	@echo "APP_BASE_URL=$(URL)" > .env || touch .env

fill-token:
	@echo "$(CALL)" >> call.txt || touch call.txt