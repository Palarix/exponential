BINARY_NAME=xpo

.PHONY: all build clean test lint frontend frontend-test install docker release-check-version release-prep release-tag stressgen setup

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
	@GOOS=windows go vet ./... 2>&1 || { echo "go vet (windows) failed"; exit 1; }
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
# Releasing is two steps so the xpo workflow does the committing and merging:
#   1. In the release issue's worktree: make release-prep VERSION=x.y.z, then
#      commit and merge the issue as usual.
#   2. On main, after the merge: make release-tag VERSION=x.y.z, then push.

release-check-version:
ifndef VERSION
	$(error VERSION is required — usage: make $(MAKECMDGOALS) VERSION=x.y.z)
endif
	@echo "$(VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$$' || { \
		echo "error: VERSION must be x.y.z, got '$(VERSION)'"; exit 1; }
	@if git rev-parse -q --verify "refs/tags/v$(VERSION)" >/dev/null; then \
		echo "error: tag v$(VERSION) already exists"; exit 1; \
	fi

# Bumps CLIVersion and turns [Unreleased] into a dated [x.y.z] section.
# perl -pi rather than sed -i, which differs between GNU and BSD.
release-prep: release-check-version
	@grep -q '^## \[Unreleased\]' CHANGELOG.md || { \
		echo "error: no [Unreleased] section found in CHANGELOG.md"; exit 1; }
	@perl -pi -e 's/^const CLIVersion = ".*"/const CLIVersion = "$(VERSION)"/' internal/version/version.go
	@perl -CSD -pi -e 's/^## \[Unreleased\]$$/## [Unreleased]\n\n## [$(VERSION)] \x{2014} $(shell date +%Y-%m-%d)/' CHANGELOG.md
	@echo "Prepared v$(VERSION): internal/version/version.go and CHANGELOG.md updated (not committed)."

# Tags the merged release on main. Does not push.
release-tag: release-check-version
	@test "$$(git rev-parse --abbrev-ref HEAD)" = main || { \
		echo "error: release-tag must run on main"; exit 1; }
	@test -z "$$(git status --porcelain)" || { \
		echo "error: working tree is not clean"; exit 1; }
	@grep -q '^const CLIVersion = "$(VERSION)"$$' internal/version/version.go || { \
		echo "error: CLIVersion is not $(VERSION) — run make release-prep in the release issue first"; exit 1; }
	@grep -q '^## \[$(VERSION)\]' CHANGELOG.md || { \
		echo "error: CHANGELOG.md has no [$(VERSION)] section"; exit 1; }
	git tag -a "v$(VERSION)" -m "v$(VERSION)"
	@echo ""
	@echo "Tagged v$(VERSION). To publish:"
	@echo "  git push origin main v$(VERSION)"
