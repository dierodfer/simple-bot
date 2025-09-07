#!make

.PHONY: clean

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

go-download: # Build the binary
	@go mod download

go-build: go-download # Build the binary
	@mkdir -p dev
	@go build -o dev/simple-bot ./cmd/simple-bot

analyze-market: go-build # Run the binary
	@go run cmd/simple-bot/main.go analyze > output.log

inspect-items: go-build
ifeq ($(strip $(INIT)), )
	$(error INIT is not set)
endif
ifeq ($(strip $(END)), )
	$(error END is not set)
endif
	@go run cmd/simple-bot/main.go inspect $(INIT) $(END) > output.log

fill-baseurl:
	@echo "APP_BASE_URL=$(URL)" > .env || touch .env

fill-token:
	@echo "$(CALL)" >> call.txt || touch call.txt