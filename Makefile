-include .env
export

DATABASE_URL ?= postgres://academy:academy@localhost:5432/academy?sslmode=disable
GOVULN  = go run golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: run worker test lint vuln build up down

run:            ## run API
	go run ./cmd/api
worker:         ## run worker
	go run ./cmd/worker
test:
	go test ./... -race -count=1
lint:
	golangci-lint run
vuln:           ## scan deps + stdlib for known vulnerabilities
	$(GOVULN) ./...
build:
	go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker
up:             ## start local Postgres
	docker compose up -d postgres
down:
	docker compose down
