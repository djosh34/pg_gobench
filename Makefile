GO := go
GOFMT := gofmt
GOFILES := $(shell find . -name '*.go' -not -path './.git/*' -not -path './vendor/*')

.PHONY: check lint test

check: lint

lint:
	@command -v $(GO) >/dev/null 2>&1 || { echo "missing required tool: $(GO)" >&2; exit 1; }
	@command -v $(GOFMT) >/dev/null 2>&1 || { echo "missing required tool: $(GOFMT)" >&2; exit 1; }
	@unformatted="$$( $(GOFMT) -l $(GOFILES) )"; \
		if [ -n "$$unformatted" ]; then \
			echo "unformatted Go files:" >&2; \
			echo "$$unformatted" >&2; \
			exit 1; \
		fi
	$(GO) vet ./...

test:
	@command -v $(GO) >/dev/null 2>&1 || { echo "missing required tool: $(GO)" >&2; exit 1; }
	$(GO) test ./...
