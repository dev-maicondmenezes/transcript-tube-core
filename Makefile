.PHONY: setup build test test-race cover fmt fmt-check vet check tidy

setup:
	go mod download

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./core/...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "arquivos não formatados:"; gofmt -l .; exit 1)

vet:
	go vet ./...

tidy:
	go mod tidy

check: fmt-check vet build test
