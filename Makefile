GO ?= $(shell which go 2>/dev/null || echo $(HOME)/.local/go/bin/go)
BINARY_NAME=veet

all: build

build:
	$(GO) build -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME)

test:
	$(GO) test -v -cover ./...

fmt:
	$(GO) fmt ./...

lint:
	$(GO) vet ./...

install: build
	mkdir -p $(HOME)/.local/bin
	cp -f $(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME) coverage.out coverage.html

help:
	@echo "Available commands:"
	@echo "  make build    - Compile the single binary"
	@echo "  make run      - Build and launch the TUI"
	@echo "  make test     - Run all unit tests"
	@echo "  make fmt      - Format all Go source files"
	@echo "  make lint     - Run go vet"
	@echo "  make install  - Install binary to GOPATH/bin"
	@echo "  make clean    - Remove build artifacts"
