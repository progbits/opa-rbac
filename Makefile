.PHONY: all build fmt

TARGET_NAME=opa-rbac

all: build

build:
	go build -o bin/$(TARGET_NAME) main.go

fmt:
	go fmt ./...

