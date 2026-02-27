BINARY_NAME := ap
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X go.agentprotocol.cloud/cli/cmd.version=$(VERSION)"

.PHONY: build install clean run tidy

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

install:
	go install $(LDFLAGS) .

clean:
	rm -rf bin/

run: build
	./bin/$(BINARY_NAME) $(ARGS)

tidy:
	go mod tidy

