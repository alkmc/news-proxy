.PHONY: build run test test-race fmt deadcode lint clean docker-build docker-up docker-down

# Variables
BINARY_NAME=news-proxy
MAIN_PATH=./cmd/newsApp/main.go

build:
	go build -o ${BINARY_NAME} ${MAIN_PATH}

run: build
	./${BINARY_NAME}

test:
	go test -v ./...

test-race:
	go test -v -race ./...

fmt:
	gofumpt -l -w .

deadcode:
	go run golang.org/x/tools/cmd/deadcode@0.47.0 ./...

lint:
	golangci-lint run

clean:
	go clean
	rm -f ${BINARY_NAME}

# Docker commands
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down
