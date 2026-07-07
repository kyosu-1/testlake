import { describe, expect, it, beforeAll } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { DuckDBInstance, type DuckDBConnection } from '@duckdb/node-api';
import { createViewsSQL, flakySQL, slowestTestsSQL, failuresSQL, buildTrendSQL } from '../src/queries';

const FIX = resolve(__dirname, 'fixtures/ci-data');
const ANCHOR = `TIMESTAMP '2026-07-08 00:00:00'`;

let conn: DuckDBConnection;

beforeAll(async () => {
  const manifest = JSON.parse(readFileSync(resolve(FIX, 'manifest.json'), 'utf8'));
  const abs = (p: string) => resolve(FIX, p);
  const inst = await DuckDBInstance.create(':memory:');
  conn = await inst.connect();
  const views = createViewsSQL(
    manifest.tables.runs.files.map((f: { path: string }) => abs(f.path)),
    manifest.tables.tests.files.map((f: { path: string }) => abs(f.path)),
  );
  for (const stmt of views.split(';').filter((s) => s.trim())) await conn.run(stmt);
});

async function rows(sql: string) {
  const reader = await conn.runAndReadAll(sql);
  return reader.getRowObjects();
}

describe('flakySQL', () => {
  it('ranks TestLogin as flaky, excludes plain failures', async () => {
    const r = await rows(flakySQL(ANCHOR));
    expect(r.length).toBe(1);
    expect(r[0].name).toBe('TestLogin');
    expect(Number(r[0].score)).toBeGreaterThan(0);
    // TestCheckout は同一 sha で pass していないので flaky ではない
    expect(r.find((x) => x.name === 'TestCheckout')).toBeUndefined();
  });
});

describe('slowestTestsSQL', () => {
  it('flags TestPay regression (100ms → ~300ms)', async () => {
    const r = await rows(slowestTestsSQL(ANCHOR));
    const pay = r.find((x) => x.name === 'TestPay');
    expect(pay).toBeDefined();
    expect(Number(pay!.ratio)).toBeGreaterThan(2);
  });
});

describe('failuresSQL', () => {
  it('reports per-workflow run counts', async () => {
    const r = await rows(failuresSQL(ANCHOR));
    const ci = r.find((x) => x.workflow === 'CI' && x.branch === 'main');
    expect(ci).toBeDefined();
    expect(Number(ci!.runs)).toBe(4);
  });
});

describe('buildTrendSQL', () => {
  it('returns one row per day/workflow/job', async () => {
    const r = await rows(buildTrendSQL(ANCHOR));
    expect(r.length).toBe(3); // 06-25, 07-06, 07-07
  });
});
