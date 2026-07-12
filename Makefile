.PHONY: build run test fmt deadcode lint clean docker-build up down

BINARY_NAME=news-proxy

build:
	go build -o ${BINARY_NAME} ./cmd/newsproxy

run: build
	./${BINARY_NAME}

test:
	go test -v -race ./...

fmt:
	gofumpt -l -w .

deadcode:
	go run golang.org/x/tools/cmd/deadcode@v0.48.0 ./...

lint:
	golangci-lint run

clean:
	go clean
	rm -f ${BINARY_NAME}

# Docker commands
docker-build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down
