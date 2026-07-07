# testlake

**CI observability with zero servers.** Turn your GitHub Actions test results into a queryable data lake — flaky test detection, test duration trends, and build analytics, served as a static page powered by DuckDB-Wasm.

> **v0.1 is under active development.** Core pieces (collector, publish action, viewer) are implemented and self-hosted (see [Usage](#usage)); expect rough edges. See the [design doc](docs/specs/2026-07-08-design.md) (Japanese).

## Why

Tools like Develocity offer great CI analytics — flaky test detection, failure analytics, build time regression — but they are enterprise-priced, and the SaaS alternatives meter you per test. There is no self-hosted OSS staple in this space.

Meanwhile, CI telemetry is *small* (megabytes per day, not gigabytes). That makes an architecture possible that doesn't work for general observability: **no query server at all**.

## How it works

```
GitHub Actions job
  └─ testlake collect (one extra step, if: always())
       ├─ parses JUnit XML reports
       ├─ collects run metadata (branch, sha, duration, runner, attempt)
       └─ appends Parquet files to your gh-pages branch

GitHub Pages
  └─ ci-data/*.parquet  +  a static viewer SPA

Your browser
  └─ DuckDB-Wasm queries the Parquet directly via HTTP range requests
```

- **Zero infrastructure.** Public repos pay nothing and run nothing. Data lives in your repo.
- **Open format.** It's just Parquet with a documented schema — query it with the bundled dashboard, the DuckDB CLI, pandas, or anything else. No lock-in by construction.
- **One-step setup:**

```yaml
- uses: kyosu-1/testlake@main
  if: always()
  with:
    reports: "test-results/**/*.xml"
```

## What you get

- **Flaky test ranking** — tests that both passed and failed on the same commit, scored by recent frequency
- **Test duration trends** — spot the test that got 2× slower last month
- **Failure analytics** — failure rates by branch and workflow
- **Build time regression** — workflow/job duration trends with anomaly highlighting
- **SQL console** — ad-hoc DuckDB SQL over `runs` and `tests`, right in the browser

## Usage

```yaml
permissions:
  contents: write
steps:
  # ... run your tests, producing JUnit XML ...
  - uses: kyosu-1/testlake@main
    if: always()
    with:
      reports: "test-results/**/*.xml"
      viewer: "true"   # deploy the dashboard to <gh-pages>/ci/
```

Then enable GitHub Pages for the `gh-pages` branch. Your dashboard lives at
`https://<user>.github.io/<repo>/ci/` and the raw Parquet at `.../ci-data/`.
See [docs/schema.md](docs/schema.md) for the storage contract.

## Roadmap

- **v0.1 (MVP):** collect action (JUnit XML → Parquet → gh-pages) + static DuckDB-Wasm viewer + published schema spec
- **v0.2+:** PR comments ("tests that got slower in this PR"), regression CI gate, private-repo backend (R2/S3 + auth), hosted viewer

## License

[MIT](LICENSE)
