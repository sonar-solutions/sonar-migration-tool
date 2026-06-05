.PHONY: build clean test install-hooks

# Build the Go binary.
build:
	cd go && go build -o sonar-migration-tool .

# Run all Go tests.
test:
	cd go && go test ./... -count=1

# Clean build artifacts.
clean:
	rm -f go/sonar-migration-tool

# Install git hooks (pre-commit secret scan via gitleaks).
install-hooks:
	sh scripts/install-git-hooks.sh
