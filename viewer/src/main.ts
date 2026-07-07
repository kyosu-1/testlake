import type { AsyncDuckDB } from '@duckdb/duckdb-wasm';
import { initDB, attachData, dataSource } from './db';
import {
  flakySQL, slowestTestsSQL, failuresSQL, buildTrendSQL, testTimelineSQL,
} from './queries';
import { renderTable, renderLine } from './render';

const TABS = [
  { id: 'flaky', label: 'Flaky tests' },
  { id: 'slow', label: 'Slow tests' },
  { id: 'failures', label: 'Failures' },
  { id: 'builds', label: 'Builds' },
  { id: 'sql', label: 'SQL' },
] as const;

let db: AsyncDuckDB;

async function q(sql: string): Promise<Record<string, unknown>[]> {
  const conn = await db.connect();
  try {
    const res = await conn.query(sql);
    return res.toArray().map((r) => r.toJSON());
  } finally {
    await conn.close();
  }
}

/**
 * duckdb-wasm の Arrow row.toJSON() は DATE/TIMESTAMP 列を数値(epoch ms)で返す(例: 1782345600000)。
 * Date インスタンスや文字列で返ってくるケースは防御的フォールバック。
 * すべてのパターンを epoch 秒に正規化する。
 */
function toEpochSec(v: unknown): number {
  if (v instanceof Date) return v.getTime() / 1000;
  if (typeof v === 'bigint') return Number(v) / 1000;
  if (typeof v === 'number') return v > 1e12 ? v / 1000 : v;
  return new Date(String(v)).getTime() / 1000;
}

async function show(tab: string, body: HTMLElement): Promise<void> {
  body.innerHTML = '<p>running…</p>';
  try {
    switch (tab) {
      case 'flaky':
        renderTable(body, await q(flakySQL()));
        break;
      case 'slow':
        renderTable(body, await q(slowestTestsSQL()));
        break;
      case 'failures':
        renderTable(body, await q(failuresSQL()));
        break;
      case 'builds': {
        const rows = await q(buildTrendSQL());
        const byKey = new Map<string, [number, number][]>();
        for (const r of rows) {
          const key = `${r.workflow} / ${r.job}`;
          const day = toEpochSec(r.day);
          (byKey.get(key) ?? byKey.set(key, []).get(key)!).push([day, Number(r.p50_duration_ms)]);
        }
        renderLine(body, [...byKey].map(([label, points]) => ({ label, points })));
        break;
      }
      case 'sql': {
        body.innerHTML = `
          <textarea id="sql-in" rows="6" style="width:100%">${testTimelineSQL}</textarea>
          <button id="sql-run">Run</button><div id="sql-out"></div>`;
        const run = async () => {
          const sql = (document.getElementById('sql-in') as HTMLTextAreaElement).value;
          const out = document.getElementById('sql-out')!;
          try {
            renderTable(out, await q(sql));
          } catch (e) {
            out.textContent = String(e);
          }
        };
        document.getElementById('sql-run')!.addEventListener('click', run);
        break;
      }
    }
  } catch (e) {
    body.innerHTML = `<p class="error">${String(e)}</p>`;
  }
}

async function boot(): Promise<void> {
  const app = document.getElementById('app')!;
  app.innerHTML = `
    <header><h1>testlake</h1><nav id="tabs"></nav></header>
    <main id="body"><p>loading DuckDB…</p></main>`;
  const nav = document.getElementById('tabs')!;
  const body = document.getElementById('body')!;
  for (const t of TABS) {
    const b = document.createElement('button');
    b.textContent = t.label;
    b.addEventListener('click', () => show(t.id, body));
    nav.appendChild(b);
  }
  try {
    db = await initDB();
    const manifest = await attachData(db, dataSource());
    if (!manifest) {
      body.innerHTML = '<p>No data yet — run your CI with the testlake action first.</p>';
      return;
    }
    await show('flaky', body);
  } catch (e) {
    body.innerHTML = `<p class="error">${String(e)}</p>`;
  }
}
boot();
