.PHONY: run build compile test

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

compile:
	forge build

test:
	forge test
