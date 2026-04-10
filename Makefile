BINARY   := dojo
MODULE   := github.com/DojoGenesis/dojo-cli
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -s -w -X main.version=$(VERSION)
GOFLAGS  := -trimpath


.PHONY: build install test vet clean fmt lint

build:
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/dojo

install:
	go install $(GOFLAGS) -ldflags '$(LDFLAGS)' ./cmd/dojo

test:
	go test ./... -count=1 -race

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

fmt:
	gofmt -s -w .

lint:
	@which golangci-lint > /dev/null 2>&1 || echo "Install golangci-lint: https://golangci-lint.run/usage/install/"
	golangci-lint run

all: vet test build
