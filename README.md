# Coverlint

![Coverage](https://raw.githubusercontent.com/evansims/coverlint/badges/coverage.svg)

Coverage checks for GitHub Actions — no external services, no secrets, no accounts. Add one step to your workflow, set a threshold, and get pass/fail results with annotations and a job summary.

Coverlint parses coverage reports in [five formats](#supported-formats), enforces configurable thresholds, and runs entirely within your GitHub Actions runner. Supports Linux, macOS, and Windows.

## Supported Formats

- **LCOV** (`lcov`) — `cargo llvm-cov`, `c8`, `istanbul`, `jest`, `vitest`
- **Go cover profile** (`gocover`) — `go test -coverprofile`
- **Cobertura XML** (`cobertura`) — `pytest-cov`, `istanbul`, `cargo tarpaulin`
- **Clover XML** (`clover`) — `phpunit`, some JS tools
- **JaCoCo XML** (`jacoco`) — Gradle/Maven JaCoCo plugin

## Usage

Add coverlint after your test step. With no inputs, it auto-detects the format, finds the report, and reports coverage without enforcing a threshold — useful for tracking trends before committing to a minimum:

```yaml
- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
```

To enforce a minimum, set `min-coverage` — a combined score across line, branch, and function coverage (see [Custom Weights](#custom-weights) for how the score is computed):

```yaml
- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    min-coverage: 80
```

Setting `format` explicitly is faster and avoids guesswork when files share names (e.g. `coverage.xml` could be Cobertura or Clover):

```yaml
- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
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
<summary><strong>Full workflow snippets</strong></summary>

**Go**

```yaml
- run: go test -coverprofile=cover.out ./...

- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: gocover
    min-coverage: 80
```

**Rust**

```yaml
- run: cargo llvm-cov --lcov --output-path lcov.info

- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: lcov
    min-coverage: 80
```

**TypeScript / JavaScript**

```yaml
- run: npx vitest run --coverage --coverage.reporter=lcov

- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: lcov
    min-coverage: 80
```

**Python**

```yaml
- run: pytest --cov --cov-report=xml:coverage.xml

- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: cobertura
    min-coverage: 80
```

**PHP**

```yaml
- run: vendor/bin/phpunit --coverage-clover=coverage.xml

- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: clover
    min-coverage: 80
```

**Java (Gradle)**

```yaml
- run: ./gradlew test jacocoTestReport

- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: jacoco
    min-coverage: 80
```

</details>

## Thresholds

### Coverage Score

`min-coverage` checks a weighted score computed from line, branch, and function coverage. The default weights are line 50, branch 30, function 20. If your format doesn't report a metric (e.g. `gocover` has no branch or function data), its weight shifts to the remaining metrics.

### Custom Weights

Weights are relative — adjust them to match what matters to your project:

```yaml
- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: lcov
    min-coverage: 80
    weight-line: 100 # only line coverage counts toward the score
    weight-branch: 0
    weight-function: 0
```

### Per-Metric Floors

Set `min-line`, `min-branch`, or `min-function` to require a minimum for a single metric, regardless of the overall score. Combine with `min-coverage` to enforce both:

```yaml
- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: lcov
    min-coverage: 80
    min-branch: 60 # fails if branch drops below 60%, even if the overall score passes
```

> [!NOTE]
> If you set a floor that your format doesn't support (e.g. `min-branch` with `gocover`), it's skipped with a warning annotation.

### Per-Area Thresholds

Use separate steps when parts of your project need different bars:

```yaml
- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: gocover
    path: cover.out
    min-coverage: 80

- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  with:
    format: lcov
    path: coverage/lcov.info
    min-coverage: 90
```

## Monorepo

Combine coverage from multiple languages in one step — the job summary breaks down each format with a combined total. Use YAML block scalars (`|`) to pass multiple values:

```yaml
- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
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

## Baseline & Regression Detection

Set `min-delta` to control how much the score can drop between runs. Coverlint compares the current score against a previous `baseline` you provide as JSON:

```yaml
- uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
  id: coverage
  with:
    format: gocover
    min-coverage: 80
    baseline: '{"score":84.5,"line":87.2,"timestamp":"2026-03-01T00:00:00Z"}'
    min-delta: 0 # no regression allowed
```

`min-delta: 0` fails on any drop. Use a negative value like `-2` to allow up to a 2-point decrease. If `baseline` is empty, delta comparison is skipped. The `baseline` output emits the current run's data as JSON, ready for storage. The full workflow below shows how to load and store it automatically.

<details>
<summary><strong>Full baseline workflow</strong></summary>

Store the baseline on an orphan branch. The test job reads the previous baseline and emits the new one; a separate job writes it back on pushes to `main`:

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
      baseline: ${{ steps.coverage.outputs.baseline }}
    steps:
      - uses: actions/checkout@v6

      # ... your test steps ...

      - name: Load previous baseline
        id: load-baseline
        env:
          REPO: ${{ github.repository }}
        run: |
          if curl -fsL "https://raw.githubusercontent.com/${REPO}/baselines/coverage-baseline.json" -o /tmp/baseline.json 2>/dev/null; then
            delimiter="COVERLINT_BASELINE_EOF"
            echo "baseline<<${delimiter}" >> "$GITHUB_OUTPUT"
            cat /tmp/baseline.json >> "$GITHUB_OUTPUT"
            echo "${delimiter}" >> "$GITHUB_OUTPUT"
          fi

      - uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
        id: coverage
        with:
          format: gocover
          min-coverage: 80
          baseline: ${{ steps.load-baseline.outputs.baseline }}
          min-delta: -2

  update-baseline:
    needs: test
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v6

      - name: Push baseline
        env:
          BASELINE: ${{ needs.test.outputs.baseline }}
        run: |
          tmpdir=$(mktemp -d)
          printf '%s' "$BASELINE" > "$tmpdir/coverage-baseline.json"

          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

          if git ls-remote --exit-code origin baselines &>/dev/null; then
            git fetch origin baselines
            git checkout baselines
          else
            git checkout --orphan baselines
            git rm -rf . 2>/dev/null || true
          fi

          cp "$tmpdir/coverage-baseline.json" .
          git add coverage-baseline.json
          git diff --cached --quiet && exit 0
          git commit -m "Update coverage baseline"
          git push origin baselines
```

</details>

## Code Scanning Integration

Generate [SARIF](https://sarifweb.azurewebsites.net/) output to surface uncovered lines and blocks in GitHub's Code Scanning tab. Set `sarif: true`, write the output to a file, and upload it with `codeql-action/upload-sarif`:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@v6

      # ... your test steps ...

      - uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
        id: coverage
        with:
          format: lcov
          sarif: true

      - name: Write SARIF file
        env:
          SARIF: ${{ steps.coverage.outputs.sarif }}
        run: printf '%s' "$SARIF" > coverage.sarif

      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: coverage.sarif
```

## PR Comments

Post coverage results as a PR comment using the `results` output:

<details>
<summary><strong>PR comment workflow</strong></summary>

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      # ... your test steps ...

      - uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
        id: coverage
        with:
          format: gocover
          min-coverage: 80

      - name: Comment on PR
        if: github.event_name == 'pull_request'
        env:
          GH_TOKEN: ${{ github.token }}
          RESULTS: ${{ steps.coverage.outputs.results }}
          PASSED: ${{ steps.coverage.outputs.passed }}
          PR_NUMBER: ${{ github.event.pull_request.number }}
        run: |
          score=$(echo "$RESULTS" | jq -r '.[-1].score // "N/A"')
          status="Pass"
          if [[ "$PASSED" != "true" ]]; then status="**Fail**"; fi

          gh pr comment "$PR_NUMBER" --body "**Coverage:** ${score}% — ${status}"
```

</details>

## Coverage Badges

Show live coverage in your README — no external services or secrets needed.

<details>
<summary><strong>Badge workflow</strong></summary>

> [!IMPORTANT]
> **Why two jobs?** The test job runs with read-only permissions on every push and PR. Only the badge job gets `contents: write`, and only on pushes to `main`. This keeps your PR checks locked down.

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

      - uses: evansims/coverlint@403f492d058d03ec2b8bee6d791a5316421dbd31 # v1.1.0
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
| `fail-on-error`     | Fail the action when thresholds are not met (default: `true`)                                          |
| `suggestions`       | Show top coverage improvement opportunities in job summary (default: `true`)                           |
| `annotations`       | Annotation output: `true` (default), `false`, or a max count                                           |
| `baseline`          | JSON string of previous baseline data for delta comparison                                             |
| `min-delta`         | Minimum allowed score change (e.g. `0` = no regression, `-2` = max 2pt drop). Ignored without `baseline` |
| `sarif`             | Generate SARIF output for GitHub Code Scanning (default: `false`)                                      |

## Outputs

| Output       | Description                                                                |
| ------------ | -------------------------------------------------------------------------- |
| `passed`     | Whether all thresholds were met (`true` or `false`)                        |
| `results`    | Coverage data as JSON                                                      |
| `badge-svg`  | Ready-to-use SVG coverage badge                                            |
| `badge-json` | Coverage badge as [shields.io](https://shields.io) endpoint JSON           |
| `baseline`   | Current run's baseline as JSON — store and feed back as the `baseline` input next run |
| `sarif`      | SARIF JSON for uploading to GitHub Code Scanning                           |

<details>
<summary><strong>Example <code>results</code> output</strong></summary>

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
- env:
    LINE: ${{ fromJSON(steps.coverage.outputs.results)[0].line }}
  run: echo "Line coverage is ${LINE}%"
```

</details>

## Exit Codes

| Code | Meaning                                   |
| ---- | ----------------------------------------- |
| 0    | All checks passed                         |
| 1    | Coverage below threshold                  |
| 2    | Configuration, parse, or unexpected error |

Use exit codes to distinguish between "tests passed but coverage is low" and "something is misconfigured." In CI, exit 1 is a meaningful failure; exit 2 usually means the action step needs fixing.

## Pinning

Releases use [immutable tags](https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases). For production workflows, [pin actions by commit SHA](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions#using-third-party-actions) and use [Dependabot](https://docs.github.com/en/code-security/dependabot/dependabot-version-updates/about-dependabot-version-updates) to keep them current. The binary is checksum-verified on every download.

## Contributing

Clone and run the tests — standard Go tooling, nothing extra needed:

```bash
git clone https://github.com/evansims/coverlint.git && cd coverlint
go test -race -cover ./...
go vet ./...
```

## License

Dual-licensed under [Apache 2.0](LICENSE-APACHE) and [MIT](LICENSE-MIT). Choose whichever you prefer.
