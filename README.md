# Coverlint

A self-contained GitHub Action that parses coverage reports, enforces thresholds, and reports results as GitHub Actions annotations and job summaries. No external service dependencies, repo secrets or other headaches to worry about — just pass/fail.

## Supported Formats

| Format           | Flag        | Typical Producer                                     |
| ---------------- | ----------- | ---------------------------------------------------- |
| LCOV             | `lcov`      | `cargo llvm-cov`, `c8`, `istanbul`, `jest`, `vitest` |
| Go cover profile | `gocover`   | `go test -coverprofile`                              |
| Cobertura XML    | `cobertura` | `pytest-cov`, `istanbul`, `cargo tarpaulin`          |
| Clover XML       | `clover`    | `phpunit`, some JS tools                             |
| JaCoCo XML       | `jacoco`    | Gradle/Maven JaCoCo plugin                           |

## Usage

Add to your workflow after your test step:

```yaml
- uses: evansims/coverlint@v1
  with:
    format: gocover
    threshold-line: 80
```

### Inputs

| Input                | Default | Required | Description                                                                                      |
| -------------------- | ------- | -------- | ------------------------------------------------------------------------------------------------ |
| `format`             |         | yes      | Coverage format(s). Comma-separated for multiple (e.g., `gocover,lcov,cobertura`)                |
| `path`               |         | no       | Path(s) to coverage files. Supports globs and comma-separated values. Auto-discovered if omitted |
| `threshold-line`     |         | no       | Minimum line coverage percentage (0-100)                                                         |
| `threshold-branch`   |         | no       | Minimum branch coverage percentage (0-100)                                                       |
| `threshold-function` |         | no       | Minimum function coverage percentage (0-100)                                                     |
| `working-directory`  | `.`     | no       | Working directory for resolving relative paths                                                   |
| `fail-on-error`      | `true`  | no       | Fail the action when thresholds are not met                                                      |
| `suggestions`        | `true`  | no       | Show top coverage improvement opportunities in job summary                                       |

When no thresholds are configured, coverlint reports coverage metrics without enforcing any minimums. This is useful for analytics and tracking coverage trends.

If a threshold is configured but the coverage format doesn't report that metric (e.g., `threshold-branch` with `gocover`), the threshold is skipped and a notice annotation is emitted.

### Auto-Discovery

When `path` is omitted, coverlint searches for coverage reports in common default locations based on the `format`:

| Format      | Searched Paths                                                                                  |
| ----------- | ----------------------------------------------------------------------------------------------- |
| `lcov`      | `coverage/lcov.info`, `lcov.info`, `coverage.lcov`                                              |
| `gocover`   | `cover.out`, `coverage.out`, `c.out`                                                            |
| `cobertura` | `coverage.xml`, `cobertura.xml`, `cobertura-coverage.xml`                                       |
| `clover`    | `coverage.xml`, `clover.xml`                                                                    |
| `jacoco`    | `build/reports/jacoco/test/jacocoTestReport.xml`, `target/site/jacoco/jacoco.xml`, `jacoco.xml` |

### Outputs

| Output    | Description                              |
| --------- | ---------------------------------------- |
| `passed`  | `true` or `false`                        |
| `results` | JSON array of per-entry coverage results |

## Examples

### Go

```yaml
- run: go test -coverprofile=cover.out ./...

- uses: evansims/coverlint@v1
  with:
    format: gocover
    threshold-line: 80
```

### Rust

```yaml
- run: cargo llvm-cov --lcov --output-path lcov.info

- uses: evansims/coverlint@v1
  with:
    path: lcov.info
    format: lcov
    threshold-line: 80
    threshold-branch: 70
```

### TypeScript / JavaScript (Vitest)

```yaml
- run: npx vitest run --coverage --coverage.reporter=lcov

- uses: evansims/coverlint@v1
  with:
    path: coverage/lcov.info
    format: lcov
    threshold-line: 80
    threshold-branch: 70
    threshold-function: 80
```

### Python (pytest)

```yaml
- run: pytest --cov --cov-report=xml:coverage.xml

- uses: evansims/coverlint@v1
  with:
    path: coverage.xml
    format: cobertura
    threshold-line: 80
    threshold-branch: 70
```

### PHP (PHPUnit)

```yaml
- run: vendor/bin/phpunit --coverage-clover=coverage.xml

- uses: evansims/coverlint@v1
  with:
    path: coverage.xml
    format: clover
    threshold-line: 80
    threshold-function: 80
```

### Java (Gradle + JaCoCo)

```yaml
- run: ./gradlew test jacocoTestReport

- uses: evansims/coverlint@v1
  with:
    path: build/reports/jacoco/test/jacocoTestReport.xml
    format: jacoco
    threshold-line: 80
    threshold-branch: 70
```

### Monorepo (Multiple Formats)

Use comma-separated `format` and `path` values to combine coverage from multiple languages in a single step. The job summary shows per-format breakdowns with a combined total.

```yaml
- uses: evansims/coverlint@v1
  with:
    format: gocover,lcov,cobertura
    path: "go-service/cover.out, node-service/coverage/lcov.info, python-service/coverage.xml"
    threshold-line: 80
```

### Multiple Independent Checks

Use separate steps when different parts of your project need different thresholds:

```yaml
- uses: evansims/coverlint@v1
  with:
    format: gocover
    path: cover.out
    threshold-line: 80

- uses: evansims/coverlint@v1
  with:
    format: lcov
    path: coverage/lcov.info
    threshold-line: 85
    threshold-branch: 70
    threshold-function: 80
```

### Analytics Only (No Thresholds)

Report coverage metrics in the job summary without enforcing any minimums:

```yaml
- uses: evansims/coverlint@v1
  with:
    format: gocover
```

## Contributing

```bash
git clone https://github.com/evansims/coverlint.git
cd coverlint
go test ./...
```

### Development

The project uses standard Go tooling:

- `go test ./...` runs all tests
- `go test -race -cover ./...` runs tests with race detection and coverage
- `go vet ./...` runs static analysis
- `go build ./cmd/coverlint` builds the binary

### Making Changes

1. Fork the repo and create a feature branch
2. Write tests for your changes
3. Run `go test ./...` and `go vet ./...`
4. Submit a pull request

### Releases

Releases are automated via GoReleaser. Pushing a version tag (e.g., `v1.0.0`) triggers cross-compilation and GitHub Release creation.

## License

Dual-licensed under [Apache 2.0](LICENSE-APACHE) and [MIT](LICENSE-MIT). Choose whichever you prefer.
