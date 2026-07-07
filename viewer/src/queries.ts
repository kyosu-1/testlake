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
