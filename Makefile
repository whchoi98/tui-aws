.PHONY: build build-all install clean test

BINARY := tui-ssm
VERSION := 0.1.0

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) ./main.go

build-all:
	GOOS=linux   GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)-linux-amd64 ./main.go
	GOOS=linux   GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)-linux-arm64 ./main.go
	GOOS=darwin  GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)-darwin-arm64 ./main.go
	GOOS=darwin  GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)-darwin-amd64 ./main.go

install:
	go build -ldflags "-X main.version=$(VERSION)" -o $(GOPATH)/bin/$(BINARY) ./main.go

test:
	go test ./... -v

clean:
	rm -f $(BINARY)
	rm -rf dist/
