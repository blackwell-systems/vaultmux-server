.PHONY: build test clean docker run

build:
	go build -o vaultmux-server ./cmd/server

test:
	go test -v ./...

clean:
	rm -f vaultmux-server
	go clean

docker:
	docker build -t vaultmux-server:latest .

run:
	go run ./cmd/server

lint:
	golangci-lint run

deps:
	go mod download
	go mod tidy
