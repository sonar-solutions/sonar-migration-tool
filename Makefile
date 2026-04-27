.PHONY: build build-web build-go clean test

# Build everything: frontend then Go binary.
build: build-web build-go

# Build the Svelte frontend into go/internal/gui/frontend/.
build-web:
	cd web && npm ci && npm run build

# Build the Go binary (requires frontend to be built first).
build-go:
	cd go && go build -o sonar-migration-tool .

# Run all Go tests.
test:
	cd go && go test ./... -count=1

# Clean build artifacts.
clean:
	rm -rf go/internal/gui/frontend/*.html go/internal/gui/frontend/*.js go/internal/gui/frontend/*.css
	rm -rf go/internal/gui/frontend/_app
	rm -rf web/.svelte-kit web/node_modules
	rm -f go/sonar-migration-tool
