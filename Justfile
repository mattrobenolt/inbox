export GOEXPERIMENT := "jsonv2"

default:
    @just --list

all: modernize fmt lint

modernize:
    go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest --fix -v ./...

lint:
    golangci-lint run

fmt: modernize
    golangci-lint fmt
    go mod tidy

build:
    go build .

run *args:
    go run . {{args}}
