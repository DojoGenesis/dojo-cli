BIN := dojo
PKG := github.com/DojoGenesis/dojo-cli/cmd/dojo

.PHONY: build run clean test tidy

build:
	go build -o $(BIN) $(PKG)

run: build
	./$(BIN)

clean:
	rm -f $(BIN)

test:
	go test ./...

tidy:
	go mod tidy

install: build
	cp $(BIN) /usr/local/bin/$(BIN)
