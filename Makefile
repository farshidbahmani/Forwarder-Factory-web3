.PHONY: run dev build compile test

run:
	go run ./cmd/server

dev:
	@command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@latest
	air

build:
	go build -o bin/server ./cmd/server

compile:
	forge build

test:
	forge test
