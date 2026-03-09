# Coverlint

![Coverage](https://raw.githubusercontent.com/evansims/coverlint/badges/coverage.svg)

A self-contained GitHub Action that parses coverage reports, enforces thresholds, and reports results as GitHub Actions annotations and job summaries. No external services or secrets required — just pass/fail.

## Supported Formats

| Format           | Flag        | Typical Producer                                     |
| ---------------- | ----------- | ---------------------------------------------------- |
| LCOV             | `lcov`      | `cargo llvm-cov`, `c8`, `istanbul`, `jest`, `vitest` |
| Go cover profile | `gocover`   | `go test -coverprofile`                              |
| Cobertura XML    | `cobertura` | `pytest-cov`, `istanbul`, `cargo tarpaulin`          |
| Clover XML       | `clover`    | `phpunit`, some JS tools                             |
| JaCoCo XML       | `jacoco`    | Gradle/Maven JaCoCo plugin                           |

## Usage

Add coverlint after your test step. With no inputs, it auto-detects the format, finds the report, and reports coverage without enforcing a threshold — useful for tracking trends before committing to a minimum:

```yaml
- uses: evansims/coverlint@v1
```

To enforce a minimum, set `min-coverage` — a combined score across line, branch, and function coverage (see [Custom Weights](#custom-weights) for how the score is computed):

```yaml
- uses: evansims/coverlint@v1
  with:
    min-coverage: 80
```

Setting `format` explicitly is faster and avoids guesswork when files share names (e.g. `coverage.xml` could be Cobertura or Clover):

```yaml
- uses: evansims/coverlint@v1
  with:
    format: lcov
    min-coverage: 80
```

## Quick Start by Language

| Language              | Test Command                                         | Format      | Path                                             |
| --------------------- | ---------------------------------------------------- | ----------- | ------------------------------------------------ |
| Go                    | `go test -coverprofile=cover.out ./...`              | `gocover`   | `cover.out`                                      |
| Rust                  | `cargo llvm-cov --lcov --output-path lcov.info`      | `lcov`      | `lcov.info`                                      |
| TypeScript/JavaScript | `npx vitest run --coverage --coverage.reporter=lcov` | `lcov`      | `coverage/lcov.info`                             |
| Python                | `pytest --cov --cov-report=xml:coverage.xml`         | `cobertura` | `coverage.xml`                                   |
| PHP                   | `vendor/bin/phpunit --coverage-clover=coverage.xml`  | `clover`    | `coverage.xml`                                   |
| Java (Gradle)         | `./gradlew test jacocoTestReport`                    | `jacoco`    | `build/reports/jacoco/test/jacocoTestReport.xml` |

<details>
<summary><strong>Go</strong></summary>

```yaml
- run: go test -coverprofile=cover.out ./...

- uses: evansims/coverlint@v1
  with:
    format: gocover
    min-coverage: 80
```

</details>

<details>
<summary><strong>Rust</strong></summary>

```yaml
- run: cargo llvm-cov --lcov --output-path lcov.info

- uses: evansims/coverlint@v1
  with:
    format: lcov
    min-coverage: 80
```

</details>

<details>
<summary><strong>TypeScript / JavaScript</strong></summary>

```yaml
- run: npx vitest run --coverage --coverage.reporter=lcov

- uses: evansims/coverlint@v1
  with:
    format: lcov
    min-coverage: 80
```

</details>

<details>
<summary><strong>Python</strong></summary>

```yaml
- run: pytest --cov --cov-report=xml:coverage.xml

- uses: evansims/coverlint@v1
  with:
    format: cobertura
    min-coverage: 80
```

</details>

<details>
<summary><strong>PHP</strong></summary>

```yaml
- run: vendor/bin/phpunit --coverage-clover=coverage.xml

- uses: evansims/coverlint@v1
  with:
    format: clover
    min-coverage: 80
```

</details>

<details>
<summary><strong>Java (Gradle)</strong></summary>

```yaml
- run: ./gradlew test jacocoTestReport

- uses: evansims/coverlint@v1
  with:
    format: jacoco
    min-coverage: 80
```

</details>

## Thresholds

### Coverage Score

`min-coverage` checks a weighted score computed from line, branch, and function coverage. The default weights are line 50, branch 30, function 20. If a metric isn't reported by your format (e.g. `gocover` doesn't report branch or function), its weight redistributes proportionally.

```yaml
- uses: evansims/coverlint@v1
  with:
    format: lcov
    min-coverage: 80
```

### Custom Weights

Weights are relative — adjust them to match what matters to your project:

```yaml
- uses: evansims/coverlint@v1
  with:
    format: lcov
    min-coverage: 80
    weight-line: 100 # only line coverage counts toward the score
    weight-branch: 0
    weight-function: 0
```

### Per-Metric Floors

Use `min-line`, `min-branch`, or `min-function` to enforce hard floors on individual metrics, checked independently of the weighted score. Combine them with `min-coverage` to set both an overall bar and individual limits:

```yaml
- uses: evansims/coverlint@v1
  with:
    format: lcov
    min-coverage: 80
    min-branch: 60 # fails if branch drops below 60%, even if the overall score passes
```

If you set a floor that your format doesn't support (e.g. `min-branch` with `gocover`), it's skipped with a warning annotation.

### Per-Area Thresholds

Use separate steps when parts of your project need different bars:

```yaml
- uses: evansims/coverlint@v1
  with:
    format: gocover
    path: cover.out
    min-coverage: 80

- uses: evansims/coverlint@v1
  with:
    format: lcov
    path: coverage/lcov.info
    min-coverage: 90
```

## Monorepo

Combine coverage from multiple languages in one step — the job summary breaks down each format with a combined total. Use YAML block scalars (`|`) to pass multiple values:

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
    min-coverage: 80
```

## Auto-Detection and Discovery

You don't need to specify `format` or `path` — coverlint can figure both out. It tries each parser until one succeeds, and looks for reports in common locations:

| Format      | Searched Paths                                                                                  |
| ----------- | ----------------------------------------------------------------------------------------------- |
| `lcov`      | `coverage/lcov.info`, `lcov.info`, `coverage.lcov`                                              |
| `gocover`   | `cover.out`, `coverage.out`, `c.out`                                                            |
| `cobertura` | `coverage.xml`, `cobertura.xml`, `cobertura-coverage.xml`                                       |
| `clover`    | `coverage.xml`, `clover.xml`                                                                    |
| `jacoco`    | `build/reports/jacoco/test/jacocoTestReport.xml`, `target/site/jacoco/jacoco.xml`, `jacoco.xml` |

## Inputs

| Input               | Description                                                                                            |
| ------------------- | ------------------------------------------------------------------------------------------------------ |
| `format`            | Coverage format(s), one per line or comma-separated. Auto-detected if omitted                          |
| `path`              | Path(s) to coverage files, one per line or comma-separated. Supports globs. Auto-discovered if omitted |
| `min-coverage`      | Minimum weighted coverage score (0-100), computed from line, branch, and function coverage             |
| `min-line`          | Minimum line coverage (0-100), checked independently of the weighted score                             |
| `min-branch`        | Minimum branch coverage (0-100), checked independently                                                 |
| `min-function`      | Minimum function coverage (0-100), checked independently                                               |
| `weight-line`       | Relative weight for line coverage in score (default: `50`)                                             |
| `weight-branch`     | Relative weight for branch coverage in score (default: `30`)                                           |
| `weight-function`   | Relative weight for function coverage in score (default: `20`)                                         |
| `working-directory` | Working directory for resolving relative paths (default: `.`)                                          |
| `fail-on-error`     | Fail the action when minimums are not met (default: `true`)                                            |
| `suggestions`       | Show top coverage improvement opportunities in job summary (default: `true`)                           |

## Outputs

| Output       | Description                                                      |
| ------------ | ---------------------------------------------------------------- |
| `passed`     | Whether all minimums were met (`true` or `false`)                |
| `results`    | Coverage data as JSON (see below)                                |
| `badge-svg`  | Ready-to-use SVG coverage badge                                  |
| `badge-json` | Coverage badge as [shields.io](https://shields.io) endpoint JSON |

The `results` JSON has one entry per format, each with a weighted `score` and available metrics. Multi-format runs include a `Total`:

```json
[
  { "name": "gocover", "score": 85, "line": 85, "passed": true },
  {
    "name": "lcov",
    "score": 77,
    "line": 78.3,
    "branch": 65.2,
    "function": 90.1,
    "passed": true
  },
  {
    "name": "Total",
    "score": 79,
    "line": 81.1,
    "branch": 65.2,
    "function": 90.1,
    "passed": true
  }
]
```

Use GitHub Actions' `fromJSON()` expression to read values in later steps:

```yaml
- run: echo "Line coverage is ${{ fromJSON(steps.coverage.outputs.results)[0].line }}%"
```

## Coverage Badges

Show live coverage in your README — no external services or secrets needed.

> **Why two jobs?** The test job runs with read-only permissions on every push and PR. Only the badge job gets `contents: write`, and only on pushes to `main`. This keeps your PR checks locked down.

<details>
<summary><strong>Badge workflow</strong></summary>

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
          min-coverage: 80

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

</details>

Add to your README:

```markdown
![Coverage](https://raw.githubusercontent.com/OWNER/REPO/badges/coverage.svg)
```

Prefer [shields.io](https://shields.io) styling? Use `badge-json` instead:

```markdown
![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/OWNER/REPO/badges/coverage.json)
```

## Pinning

Releases use [immutable tags](https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases). For production workflows, [pin actions by commit SHA](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions#using-third-party-actions) and use [Dependabot](https://docs.github.com/en/code-security/dependabot/dependabot-version-updates/about-dependabot-version-updates) to keep them current.

## Contributing

Clone and run the tests — standard Go tooling, nothing extra needed:

```bash
git clone https://github.com/evansims/coverlint.git && cd coverlint
go test -race -cover ./...
go vet ./...
```

## License

Dual-licensed under [Apache 2.0](LICENSE-APACHE) and [MIT](LICENSE-MIT). Choose whichever you prefer.
