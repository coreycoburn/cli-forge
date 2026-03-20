.PHONY: build install test clean

# Build all CLIs to ./bin/
build:
	@mkdir -p bin
	@for dir in cmd/*/; do \
		name=$$(basename "$$dir"); \
		echo "Building $$name..."; \
		go build -o bin/$$name ./cmd/$$name; \
	done

# Install all CLIs to $GOPATH/bin
install:
	@for dir in cmd/*/; do \
		name=$$(basename "$$dir"); \
		echo "Installing $$name..."; \
		go install ./cmd/$$name; \
	done

# Run tests
test:
	go test ./...

# Remove build artifacts
clean:
	rm -rf bin/ dist/
