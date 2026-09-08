BINARY_NAME=xpo

.PHONY: all build clean test lint frontend frontend-test install docker release stressgen setup

all: setup build

# Build the frontend with Vite and copy to Go static folder
frontend:
	@echo "Building web UI..."
	@cd web && bun install --silent && bun run build 2>&1 | tail -1
	@rm -rf internal/server/static
	@cp -r web/dist internal/server/static
	@touch internal/server/static/.gitkeep

# Build the Go binary (depends on frontend)
build: test frontend cli

# Build Go binary only (skip frontend rebuild)
cli:
	@echo "Building CLI..."
	@go build -o $(BINARY_NAME) ./cmd/exponential

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -rf web/dist
	rm -rf internal/server/static
	mkdir -p internal/server/static
	touch internal/server/static/.gitkeep

test: lint frontend-test
	@echo "Running test suite..."
	@go test ./... > /dev/null 2>&1 || go test -v ./...

frontend-test:
	@echo "Running frontend tests..."
	@cd web && bun run --silent test 2>&1 || { echo "vitest failed"; exit 1; }

lint:
	@echo "Running lint..."
	@go vet ./... 2>&1 || { echo "go vet failed"; exit 1; }
	@cd web && bun run --silent lint 2>&1 || { echo "eslint failed"; exit 1; }

install: cli
	sudo rm -f /usr/local/bin/xpo
	sudo cp ./xpo /usr/local/bin/xpo

setup:
	@git config core.hooksPath .githooks

stressgen:
	go run ./tools/stressgen $(or $(OUTPUT),stresstest)

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
