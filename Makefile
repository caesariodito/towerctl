BIN := towerctl

.PHONY: build test run clean

build:
	go build -o bin/$(BIN) ./cmd/towerctl

test:
	go test ./...

run:
	go run ./cmd/towerctl --help

clean:
	rm -rf bin
