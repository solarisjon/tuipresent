VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short=7 HEAD 2>/dev/null)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
PKG     := ./cmd/tuipresent
TARGETS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: build test check-examples release clean

build:
	go build -ldflags '$(LDFLAGS)' -o bin/tuipresent $(PKG)

test:
	gofmt -l . | (! grep .)
	go vet ./...
	go test -count=1 ./...

# Runs -check (never -exec) on every example deck at both reference sizes.
check-examples: build
	@set -e; for f in examples/*.md; do \
		case $$f in examples/README.md) continue;; esac; \
		bin/tuipresent -check -w 100 -h 30 $$f; \
		bin/tuipresent -check -w 80 -h 24 $$f; \
	done

# Cross-compiles every target into dist/ and writes dist/SHA256SUMS.
# Override the version: make release VERSION=v1.0.0-rc1
release:
	rm -rf dist
	mkdir -p dist
	@set -e; for t in $(TARGETS); do \
		os=$${t%/*}; arch=$${t#*/}; \
		echo "building tuipresent-$(VERSION)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags '$(LDFLAGS)' \
			-o dist/tuipresent-$(VERSION)-$$os-$$arch $(PKG); \
	done
	cd dist && (command -v sha256sum >/dev/null && sha256sum tuipresent-* || shasum -a 256 tuipresent-*) > SHA256SUMS
	@cat dist/SHA256SUMS

clean:
	rm -rf bin dist
