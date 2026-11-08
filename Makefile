-include .env
export

DATABASE_URL ?= postgres://academy:academy@localhost:5432/academy?sslmode=disable
GOVULN  = go run golang.org/x/vuln/cmd/govulncheck@latest
GOOSE   = go run github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir migrations postgres "$(DATABASE_URL)"
SQLC    = go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

.PHONY: run test lint vuln build sqlc migrate-up migrate-down migrate-status migrate-new

run:            ## run API
	go run ./cmd/api
test:           ## set TEST_DATABASE_URL to also run the migration tests
	go test ./... -race -count=1
lint:
	golangci-lint run
vuln:           ## scan deps + stdlib for known vulnerabilities
	$(GOVULN) ./...
build:
	go build -o bin/api ./cmd/api
sqlc:           ## regenerate internal/*/store from migrations/ and internal/*/queries/
	$(SQLC) generate
migrate-up:     ## apply pending migrations
	$(GOOSE) up
migrate-down:   ## roll back the last migration
	$(GOOSE) down
migrate-status:
	$(GOOSE) status
migrate-new:    ## make migrate-new name=add_foo
	go run github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir migrations create $(name) sql
