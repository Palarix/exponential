BINARY_NAME=beats

.PHONY: all build clean test lint frontend install

all: build

# Build the frontend with Vite and copy to Go static folder
frontend:
	cd web && bun run build
	rm -rf internal/server/static
	cp -r web/dist internal/server/static

# Build the Go binary (depends on frontend)
build: frontend cli

# Build Go binary only (skip frontend rebuild)
cli:
	go build -o $(BINARY_NAME) ./cmd/beats

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
	sudo cp ./beats /usr/local/bin/beats
