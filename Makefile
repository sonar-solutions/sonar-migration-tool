.PHONY: build clean test

# Build the Go binary.
build:
	cd go && go build -o sonar-migration-tool .

# Run all Go tests.
test:
	cd go && go test ./... -count=1

# Clean build artifacts.
clean:
	rm -f go/sonar-migration-tool
