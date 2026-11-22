.PHONY: build run test fmt deadcode lint fix dockerize up down

build:
	go build ./cmd/newsproxy

run:
	go run ./cmd/newsproxy

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
