.PHONY: build run test fmt deadcode lint fix dockerize up down

BINARY_NAME=news-proxy

build:
	go build -o ${BINARY_NAME} ./cmd/newsproxy

run: build
	./${BINARY_NAME}

test:
	go test -v -race ./...

fmt:
	golangci-lint fmt

deadcode:
	go run golang.org/x/tools/cmd/deadcode@v0.53.0 -test ./...

lint:
	golangci-lint run

fix: 
	go fix ./... && golangci-lint run --fix

dockerize:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down
