.PHONY: all build deps fmt

TARGET_NAME=opa-rbac

all: build

build: deps
	go build -o bin/$(TARGET_NAME) main.go

deps:
	go mod download

fmt:
	go fmt ./...

