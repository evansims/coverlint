# Coverlint

![Coverage](https://raw.githubusercontent.com/evansims/coverlint/badges/coverage.svg)

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
    format: gocover       # recommended; auto-detected if omitted
    threshold-line: 80
```

### Inputs

| Input                | Default | Description                                                                                           |
| -------------------- | ------- | ----------------------------------------------------------------------------------------------------- |
| `format`             |         | Coverage format(s), one per line or comma-separated. Auto-detected if omitted                         |
| `path`               |         | Path(s) to coverage files, one per line or comma-separated. Supports globs. Auto-discovered if omitted |
| `threshold-line`     |         | Minimum line coverage percentage (0-100)                                                              |
| `threshold-branch`   |         | Minimum branch coverage percentage (0-100)                                                            |
| `threshold-function` |         | Minimum function coverage percentage (0-100)                                                          |
| `working-directory`  | `.`     | Working directory for resolving relative paths                                                        |
| `fail-on-error`      | `true`  | Fail the action when thresholds are not met                                                           |
| `suggestions`        | `true`  | Show top coverage improvement opportunities in job summary                                            |

When no thresholds are configured, coverlint reports coverage metrics without enforcing any minimums. This is useful for analytics and tracking coverage trends.

If a threshold is configured but the coverage format doesn't report that metric (e.g., `threshold-branch` with `gocover`), the threshold is skipped and a notice annotation is emitted.

### Auto-Detection

When `format` is omitted, coverlint tries each parser in priority order (gocover, lcov, jacoco, cobertura, clover) against each discovered file until one succeeds. Specifying `format` explicitly is recommended — it's faster and avoids ambiguity when files could match multiple formats (e.g., `coverage.xml` could be Cobertura or Clover).

### Auto-Discovery

When `path` is omitted, coverlint searches for coverage reports in common default locations. If `format` is also omitted, it scans all known default paths across all formats:

| Format      | Searched Paths                                                                                  |
| ----------- | ----------------------------------------------------------------------------------------------- |
| `lcov`      | `coverage/lcov.info`, `lcov.info`, `coverage.lcov`                                              |
| `gocover`   | `cover.out`, `coverage.out`, `c.out`                                                            |
| `cobertura` | `coverage.xml`, `cobertura.xml`, `cobertura-coverage.xml`                                       |
| `clover`    | `coverage.xml`, `clover.xml`                                                                    |
| `jacoco`    | `build/reports/jacoco/test/jacocoTestReport.xml`, `target/site/jacoco/jacoco.xml`, `jacoco.xml` |

### Outputs

| Output       | Description                                      |
| ------------ | ------------------------------------------------ |
| `passed`     | `true` or `false`                                |
| `results`    | JSON array of per-entry coverage results         |
| `badge-svg`  | SVG badge showing line coverage percentage       |
| `badge-json` | Shields.io endpoint JSON for line coverage badge |

The `results` output is a JSON array you can parse in subsequent steps:

```json
[
  {
    "name": "gocover",
    "line": 82.5,
    "passed": true
  }
]
```

For multi-format runs, each format gets its own entry plus a combined total:

```json
[
  { "name": "gocover", "line": 85.0, "passed": true },
  { "name": "lcov", "line": 78.3, "branch": 65.2, "function": 90.1, "passed": true },
  { "name": "Total", "line": 81.1, "branch": 65.2, "function": 90.1, "passed": true }
]
```

Fields like `branch` and `function` are omitted when the format doesn't report them. Use `fromJSON()` to access these values:

```yaml
- uses: evansims/coverlint@v1
  id: coverage
  with:
    format: gocover

- run: echo "Line coverage is ${{ fromJSON(steps.coverage.outputs.results)[0].line }}%"
```

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

List multiple formats and paths to combine coverage from different languages in a single step. The job summary shows per-format breakdowns with a combined total.

```yaml
- uses: evansims/coverlint@v1
  with:
    format: |
      gocover
      lcov
      cobertura
    path: |
      go-service/cover.out
      node-service/coverage/lcov.info
      python-service/coverage.xml
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

### Fully Automatic

When both `format` and `path` are omitted, coverlint discovers files from known default locations and auto-detects the format:

```yaml
- uses: evansims/coverlint@v1
  with:
    threshold-line: 80
```

## Coverage Badges

Coverlint generates badge outputs that you can use to display live coverage in your README. No external services or secrets required.

Use a two-job workflow to follow the principle of least privilege — the test job runs with read-only permissions on every push and PR, while a separate badge job only runs on `main` with the `contents: write` permission it needs:

```yaml
on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    outputs:
      badge-svg: ${{ steps.coverage.outputs.badge-svg }}
    steps:
      - uses: actions/checkout@v6

      # ... your test steps ...

      - uses: evansims/coverlint@v1
        id: coverage
        with:
          format: gocover
          threshold-line: 80

  update-badges:
    needs: test
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v6

      - name: Push coverage badge
        env:
          BADGE_SVG: ${{ needs.test.outputs.badge-svg }}
        run: |
          tmpdir=$(mktemp -d)
          printf '%s' "$BADGE_SVG" > "$tmpdir/coverage.svg"

          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

          if git ls-remote --exit-code origin badges &>/dev/null; then
            git fetch origin badges
            git checkout badges
          else
            git checkout --orphan badges
            git rm -rf . 2>/dev/null || true
          fi

          cp "$tmpdir/coverage.svg" .
          git add coverage.svg
          git diff --cached --quiet && exit 0
          git commit -m "Update coverage badge"
          git push origin badges
```

Then reference the badge in your README:

```markdown
![Coverage](https://raw.githubusercontent.com/OWNER/REPO/badges/coverage.svg)
```

### Using shields.io instead

If you prefer shields.io styling, write the `badge-json` output instead and use a shields.io endpoint badge:

```markdown
![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/OWNER/REPO/badges/coverage.json)
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
