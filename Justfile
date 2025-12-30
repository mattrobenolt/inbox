export GOEXPERIMENT := "jsonv2"

default:
    @just --list

all: modernize fmt lint

modernize:
    go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest --fix -v ./...

lint:
    golangci-lint run

fmt:
    golangci-lint fmt

build:
    go build .

run *args:
    go run . {{args}}
