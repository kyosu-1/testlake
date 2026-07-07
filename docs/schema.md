# testlake storage layout (schema_version = 1)

## Directory layout

```
ci-data/
  manifest.json
  runs/date=YYYY-MM-DD/<run_id>-<job_slug>-<attempt>.parquet
  tests/date=YYYY-MM-DD/<run_id>-<job_slug>-<attempt>.parquet
```

Note: During compaction, a day's individual files may be replaced with a single `compacted.parquet`.

## runs table

| column | type | description |
|---|---|---|
| schema_version | INT32 | always 1 |
| run_id | INT64 | GITHUB_RUN_ID |
| run_attempt | INT32 | GITHUB_RUN_ATTEMPT |
| workflow | UTF8 | Actions context: workflow name |
| job | UTF8 | Actions context: job name |
| branch | UTF8 | Actions context: branch name |
| sha | UTF8 | Actions context: commit SHA |
| event | UTF8 | Actions context: event type |
| runner_os | UTF8 | Actions context: runner OS |
| started_at | TIMESTAMP(ms, UTC) | job start time (GitHub API); collection time as fallback |
| duration_ms | INT64 | job duration in milliseconds; 0 = unknown |
| conclusion | UTF8 | job.status: `success`, `failure`, `cancelled`, `unknown` |

## tests table

| column | type | description |
|---|---|---|
| schema_version | INT32 | always 1 |
| run_id | INT64 | denormalized from runs: GITHUB_RUN_ID |
| run_attempt | INT32 | denormalized from runs: GITHUB_RUN_ATTEMPT |
| job | UTF8 | denormalized from runs: job name |
| branch | UTF8 | denormalized from runs: branch name |
| sha | UTF8 | denormalized from runs: commit SHA |
| started_at | TIMESTAMP(ms, UTC) | denormalized from runs: job start time |
| suite | UTF8 | from JUnit XML: test suite name |
| class | UTF8 | from JUnit XML: test class name |
| name | UTF8 | from JUnit XML: test case name |
| file | UTF8 | from JUnit XML: file path |
| outcome | UTF8 | test result: `passed`, `failed`, `error`, `skipped` |
| duration_ms | INT64 | test duration in milliseconds |
| failure_message | UTF8 | failure description; truncated to 4096 bytes |
| failure_type | UTF8 | failure type or category |

## manifest.json

The manifest describes all data files and enables efficient listing without directory crawling:

```json
{
  "schema_version": 1,
  "generated_at": "ISO8601 timestamp",
  "tables": {
    "runs": {
      "files": [
        {"path": "runs/date=2024-01-01/123-workflow-1.parquet", "rows": 42, "bytes": 8192},
        ...
      ]
    },
    "tests": {
      "files": [
        {"path": "tests/date=2024-01-01/123-workflow-1.parquet", "rows": 1250, "bytes": 65536},
        ...
      ]
    }
  }
}
```

**Important:** Readers should list files from the manifest.json rather than crawling the directory tree. This ensures consistency and better performance.

## Retention and Compaction

- Default retention: 400 days
- Parquet files for a single day may be consolidated into a `compacted.parquet` file during maintenance
- Readers must handle both individual and compacted files

## Versioning

- `schema_version` is included in every row and in manifest.json
- Backward-incompatible changes bump `schema_version` (major version only; this is schema version 1)
- Readers **must ignore unknown columns** to maintain forward compatibility
- Forward-compatible additions (new columns) do not change the schema version

## Example query

```bash
duckdb -c "SELECT outcome, count(*) FROM read_parquet('https://<user>.github.io/<repo>/ci-data/tests/*/*.parquet') GROUP BY 1"
```

This query works across all test files, aggregating results by outcome.
