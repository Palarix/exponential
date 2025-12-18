BINARY_NAME=beats

.PHONY: all build clean test lint

all: build

build:
	go build -o $(BINARY_NAME) ./cmd/beats

clean:
	go clean
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

lint:
	go vet ./...
	# Assuming staticcheck is installed, if not, user might need to install it
	# staticcheck ./...
