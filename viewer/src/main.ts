import { initDB, attachData, dataSource } from './db';

async function boot() {
  const app = document.getElementById('app')!;
  try {
    const db = await initDB();
    const manifest = await attachData(db, dataSource());
    app.textContent = manifest ? `data loaded (schema v${manifest.schema_version})` : 'no data yet';
  } catch (e) {
    app.textContent = String(e);
  }
}
boot();
