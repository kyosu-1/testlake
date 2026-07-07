import * as duckdb from '@duckdb/duckdb-wasm';
import duckdb_wasm from '@duckdb/duckdb-wasm/dist/duckdb-mvp.wasm?url';
import mvp_worker from '@duckdb/duckdb-wasm/dist/duckdb-browser-mvp.worker.js?url';
import duckdb_wasm_eh from '@duckdb/duckdb-wasm/dist/duckdb-eh.wasm?url';
import eh_worker from '@duckdb/duckdb-wasm/dist/duckdb-browser-eh.worker.js?url';
import { createViewsSQL } from './queries';

export interface Manifest {
  schema_version: number;
  generated_at: string;
  tables: Record<'runs' | 'tests', { files: { path: string; rows: number; bytes: number }[] }>;
}

const BUNDLES: duckdb.DuckDBBundles = {
  mvp: { mainModule: duckdb_wasm, mainWorker: mvp_worker },
  eh: { mainModule: duckdb_wasm_eh, mainWorker: eh_worker },
};

export async function initDB(): Promise<duckdb.AsyncDuckDB> {
  const bundle = await duckdb.selectBundle(BUNDLES);
  const worker = new Worker(bundle.mainWorker!);
  const db = new duckdb.AsyncDuckDB(new duckdb.ConsoleLogger(duckdb.LogLevel.WARNING), worker);
  await db.instantiate(bundle.mainModule);
  return db;
}

/** manifest.json を読み、runs/tests VIEW を張る。データが空なら null を返す。 */
export async function attachData(db: duckdb.AsyncDuckDB, srcBase: string): Promise<Manifest | null> {
  const base = srcBase.endsWith('/') ? srcBase : srcBase + '/';
  const res = await fetch(base + 'manifest.json', { cache: 'no-cache' });
  if (!res.ok) throw new Error(`manifest.json: HTTP ${res.status}`);
  const manifest: Manifest = await res.json();
  const abs = (p: string) => new URL(p, base).toString();
  const runFiles = manifest.tables.runs.files.map((f) => abs(f.path));
  const testFiles = manifest.tables.tests.files.map((f) => abs(f.path));
  if (runFiles.length === 0 || testFiles.length === 0) return null;
  const conn = await db.connect();
  try {
    for (const stmt of createViewsSQL(runFiles, testFiles).split(';').filter((s) => s.trim())) {
      await conn.query(stmt);
    }
  } finally {
    await conn.close();
  }
  return manifest;
}

/** ?src= 指定がなければ、ビューアが <pages>/ci/ に居る前提で ../ci-data/ を読む */
export function dataSource(): string {
  const raw = new URLSearchParams(location.search).get('src');
  if (raw !== null) return new URL(raw, location.href).toString();
  return new URL('../ci-data/', location.href).toString();
}
