// All aggregation logic lives in this file as SQL. One panel = one query.
// Every query takes an `anchor` timestamp expression (default: now()) so
// tests can pin "today" and stay deterministic forever.

export function createViewsSQL(runFiles: string[], testFiles: string[]): string {
  const list = (fs: string[]) => fs.map((f) => `'${f.replaceAll("'", "''")}'`).join(', ');
  return `
    CREATE OR REPLACE VIEW runs  AS SELECT * FROM read_parquet([${list(runFiles)}], union_by_name=true);
    CREATE OR REPLACE VIEW tests AS SELECT * FROM read_parquet([${list(testFiles)}], union_by_name=true);
  `;
}

export function flakySQL(anchor = 'now()'): string {
  return `
    WITH recent AS (
      SELECT * FROM tests
      WHERE started_at > ${anchor} - INTERVAL 30 DAY
        AND outcome IN ('passed','failed','error')
    ),
    per_sha AS (
      SELECT class, name, sha,
             max(started_at) AS last_seen,
             count(*) FILTER (outcome = 'passed') AS passes,
             count(*) FILTER (outcome IN ('failed','error')) AS fails
      FROM recent GROUP BY ALL
    )
    SELECT class, name,
           sum(exp(-date_diff('day', last_seen, ${anchor}) / 7.0)) AS score,
           count(*) AS flaky_shas,
           max(last_seen) AS last_flake
    FROM per_sha
    WHERE passes > 0 AND fails > 0
    GROUP BY ALL
    ORDER BY score DESC`;
}

export function slowestTestsSQL(anchor = 'now()'): string {
  return `
    WITH passed AS (
      SELECT * FROM tests WHERE outcome = 'passed'
        AND started_at > ${anchor} - INTERVAL 37 DAY
    ),
    recent AS (
      SELECT class, name, median(duration_ms) AS p50_recent_ms
      FROM passed WHERE started_at > ${anchor} - INTERVAL 7 DAY GROUP BY ALL
    ),
    prior AS (
      SELECT class, name, median(duration_ms) AS p50_prior_ms
      FROM passed WHERE started_at <= ${anchor} - INTERVAL 7 DAY GROUP BY ALL
    )
    SELECT r.class, r.name, r.p50_recent_ms, p.p50_prior_ms,
           r.p50_recent_ms / nullif(p.p50_prior_ms, 0) AS ratio
    FROM recent r JOIN prior p USING (class, name)
    ORDER BY ratio DESC NULLS LAST
    LIMIT 50`;
}

export function failuresSQL(anchor = 'now()'): string {
  return `
    SELECT workflow, branch,
           count(*) AS runs,
           count(*) FILTER (conclusion = 'failure') AS failed_runs,
           count(*) FILTER (conclusion = 'failure') / count(*)::DOUBLE AS failure_rate
    FROM runs
    WHERE started_at > ${anchor} - INTERVAL 30 DAY
    GROUP BY ALL
    ORDER BY failure_rate DESC, runs DESC`;
}

export function buildTrendSQL(anchor = 'now()'): string {
  return `
    SELECT date_trunc('day', started_at)::DATE AS day, workflow, job,
           median(duration_ms) AS p50_duration_ms
    FROM runs
    WHERE started_at > ${anchor} - INTERVAL 90 DAY
    GROUP BY ALL
    ORDER BY day`;
}

export const testTimelineSQL = `SELECT * FROM tests ORDER BY started_at DESC LIMIT 100`;
