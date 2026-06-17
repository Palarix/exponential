BINARY_NAME=xpo

.PHONY: all build clean test lint frontend install docker release

all: build

# Build the frontend with Vite and copy to Go static folder
frontend:
	cd web && bun install && bun run build
	rm -rf internal/server/static
	cp -r web/dist internal/server/static

# Build the Go binary (depends on frontend)
build: frontend cli

# Build Go binary only (skip frontend rebuild)
cli:
	go build -o $(BINARY_NAME) ./cmd/exponential

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -rf web/dist
	rm -rf internal/server/static

test:
	go test -v ./...

lint:
	go vet ./...
	# Assuming staticcheck is installed, if not, user might need to install it
	# staticcheck ./...

install: cli
	sudo cp ./xpo /usr/local/bin/xpo

docker:
	docker build -t palarix/exponential:latest .

# Cut a release: make release VERSION=x.y.z
release:
ifndef VERSION
	$(error VERSION is required — usage: make release VERSION=x.y.z)
endif
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "error: working tree is not clean — commit or stash changes first"; exit 1; \
	fi
	@if git rev-parse "v$(VERSION)" >/dev/null 2>&1; then \
		echo "error: tag v$(VERSION) already exists"; exit 1; \
	fi
	@if ! grep -q '## \[Unreleased\]' CHANGELOG.md; then \
		echo "error: no [Unreleased] section found in CHANGELOG.md"; exit 1; \
	fi
	@echo "Releasing v$(VERSION)..."
	sed -i 's/const CLIVersion = ".*"/const CLIVersion = "$(VERSION)"/' internal/version/version.go
	sed -i 's/## \[Unreleased\]/## [Unreleased]\n\n## [$(VERSION)] - $(shell date +%Y-%m-%d)/' CHANGELOG.md
	git add internal/version/version.go CHANGELOG.md
	git commit -m "release: v$(VERSION)"
	git tag -a "v$(VERSION)" -m "v$(VERSION)"
	@echo ""
	@echo "Done. To publish:"
	@echo "  git push origin main v$(VERSION)"
