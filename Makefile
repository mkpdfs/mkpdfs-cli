.PHONY: build clean test test-integration dev-link fmt lint

# Local dev/test binary is `mkp-cli` so it never shadows the production `mkp`
# installed from Homebrew. The release binary (named `mkp`) is built by GoReleaser
# (.goreleaser.yml), not this Makefile.
VERSION ?= 0.1.0
BINARY_NAME = mkp-cli
LDFLAGS = -ldflags "-s -w -X github.com/sim4gh/mkpdfs-cli/internal/cli.Version=$(VERSION)"

build:
	@go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/mkp

dev-link: build
	@ln -sf "$(PWD)/$(BINARY_NAME)" /opt/homebrew/bin/$(BINARY_NAME); \
	echo "Linked $(BINARY_NAME) -> /opt/homebrew/bin/$(BINARY_NAME)"

test:
	@go test ./...

test-integration:
	@go test -tags=integration -timeout=5m -count=1 ./test/integration/

fmt:
	@go fmt ./...

lint:
	@golangci-lint run

clean:
	@rm -f $(BINARY_NAME)
