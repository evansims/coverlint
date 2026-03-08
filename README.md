# Coverage

A self-contained GitHub Action that enforces coverage thresholds on pull requests. Parses coverage reports, compares against configurable thresholds, and reports results as GitHub Actions annotations and job summaries.

No external services. No GitHub API tokens. No PR comments. Just pass/fail.

## Supported Formats

| Format | Flag | Typical Producer |
|--------|------|------------------|
| LCOV | `lcov` | `cargo llvm-cov`, `c8`, `istanbul`, `jest`, `vitest` |
| Go cover profile | `gocover` | `go test -coverprofile` |
| Cobertura XML | `cobertura` | `pytest-cov`, `istanbul`, `cargo tarpaulin` |
| Clover XML | `clover` | `phpunit`, some JS tools |
| JaCoCo XML | `jacoco` | Gradle/Maven JaCoCo plugin |

## Installation

Add to your workflow after your test step:

```yaml
- uses: evansims/coverage@v1
```

The action reads `coverage.json` from your repo root by default.

### Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `config` | `coverage.json` | Path to config file, relative to working directory |
| `working-directory` | `.` | Working directory for resolving relative paths |
| `fail-on-error` | `true` | Fail the action when thresholds are not met |

### Outputs

| Output | Description |
|--------|-------------|
| `passed` | `true` or `false` |
| `results` | JSON array of per-entry coverage results |

## Configuration

Create a `coverage.json` in your repo root:

```json
{
  "version": 1,
  "coverage": [
    {
      "name": "backend",
      "path": "coverage/lcov.info",
      "format": "lcov",
      "threshold": {
        "line": 80,
        "branch": 70,
        "function": 80
      }
    }
  ]
}
```

### Schema

**Top-level:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | integer | yes | Schema version, currently `1` |
| `coverage` | array | yes | List of coverage entries |

**Each coverage entry:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Display name for annotations |
| `path` | string | yes | Path to coverage report file |
| `format` | string | yes | One of: `lcov`, `gocover`, `cobertura`, `clover`, `jacoco` |
| `threshold` | object | yes | At least one threshold must be set |

**Threshold fields (all optional, but at least one required):**

| Field | Type | Range | Description |
|-------|------|-------|-------------|
| `line` | number | 0-100 | Minimum line coverage percentage |
| `branch` | number | 0-100 | Minimum branch coverage percentage |
| `function` | number | 0-100 | Minimum function coverage percentage |

If a threshold is configured but the coverage format doesn't report that metric (e.g., `branch` with `gocover`), the threshold is skipped and a notice annotation is emitted.

## Example Workflow

```yaml
name: Coverage
on: [pull_request]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - run: go test -coverprofile=cover.out ./...

      - uses: evansims/coverage@v1
        with:
          fail-on-error: 'true'
```

### Multiple Reports

```json
{
  "version": 1,
  "coverage": [
    {
      "name": "api",
      "path": "cover.out",
      "format": "gocover",
      "threshold": { "line": 80 }
    },
    {
      "name": "frontend",
      "path": "coverage/lcov.info",
      "format": "lcov",
      "threshold": { "line": 85, "branch": 70, "function": 80 }
    }
  ]
}
```

## Contributing

```bash
git clone https://github.com/evansims/coverage.git
cd coverage
go test ./...
```

### Development

The project uses standard Go tooling:

- `go test ./...` runs all tests
- `go test -race -cover ./...` runs tests with race detection and coverage
- `go vet ./...` runs static analysis
- `go build ./cmd/coverage` builds the binary

### Making Changes

1. Fork the repo and create a feature branch
2. Write tests for your changes
3. Run `go test ./...` and `go vet ./...`
4. Submit a pull request

### Releases

Releases are automated via GoReleaser. Pushing a version tag (e.g., `v1.0.0`) triggers cross-compilation and GitHub Release creation.

## License

Dual-licensed under [Apache 2.0](LICENSE-APACHE) and [MIT](LICENSE-MIT). Choose whichever you prefer.
