# Интеграционным тестам нужна база со схемой из миграций Django:
#   TEST_DATABASE_URL=postgres://outlove@127.0.0.1:5432/outlove_test?sslmode=disable make test-integration
GO ?= go

.PHONY: build test test-integration vet run

build:
	$(GO) build -o bin/api ./cmd/api

test:
	$(GO) test ./...

# -p 1: пакеты используют одну базу и очищают её перед тестом
test-integration:
	$(GO) test -tags integration -p 1 -count=1 ./...

vet:
	$(GO) vet ./...
	$(GO) vet -tags integration ./...

run:
	$(GO) run ./cmd/api
