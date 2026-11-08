# Repository Guidelines

## Project Structure & Module Organization

`bam` is a standard-library-only Go client for BlueCat Address Manager RESTful v2. The root package owns transport, authentication, pagination, and lossless JSON documents. Domain APIs live in `configuration/`, `dns/` (including `accesscontrol/` and `responsepolicy/`), `ipam/`, `dhcp/`, and `deployment/`. Shared collection helpers live in `internal/resource/`. Tests sit beside source files; `docs/` contains architecture decisions and capability evidence, and `scripts/` handles packaging and releases.

## Build, Test, and Development Commands

Run from the repository root with Go 1.23 or later; `go.mod` specifies the preferred toolchain.

- `go build ./...`: compile all library packages.
- `go test -race -shuffle=on -count=1 -timeout 60s ./...`: run the CI test suite.
- `go vet ./...`: run Go static checks.
- `golangci-lint run`: apply `.golangci.yml` checks.
- `govulncheck ./...`: check reachable vulnerabilities.
- `GOTOOLCHAIN=go1.23.0 GOWORK=off go test -timeout 60s ./...`: verify minimum-version compatibility.

CI also checks formatting and requires `go mod tidy` to leave tracked files unchanged.

## Coding Style & Naming Conventions

Format edited Go files with `gofmt` (tab indentation). Use lowercase package names, MixedCaps identifiers, and descriptive filenames such as `record_query.go`. Document exported APIs. Put `context.Context` first in network operations, follow existing `Spec`/`Changes` patterns, and preserve the standard-library-only runtime.

## Testing Guidelines

Use standard `testing` and `net/http/httptest`. Place `TestXxx` and executable `ExampleXxx` functions in adjacent `*_test.go` files, usually in `<package>_test`. Exercise public HTTP contracts, errors, cancellation, and field preservation without appliance credentials. CI has no numeric coverage gate; local contract tests do not establish live-appliance acceptance.

## Commit & Pull Request Guidelines

Follow existing `feat:`, `fix:`, and `feat!:` subjects; mark breaking API changes. PRs should describe changed behavior, link relevant issues, list validation commands/results, and explain compatibility effects. Update `README.md`, `CHANGELOG.md`, and `docs/rest-v2-coverage.md` when public behavior or supported operations change.

## Architecture & Safety

For domain changes, consult [the glossary](GLOSSARY.md) and [ADR 0002](docs/adr/0002-typed-domain-capabilities.md). Preserve unknown JSON fields and distinguish unchanged, zero, and null values. Keep deployment submission explicit and writes free of automatic retries. Keep TLS verification enabled and credentials out of tests and logs.
