import uPlot from 'uplot';
import 'uplot/dist/uPlot.min.css';

export function renderTable(el: HTMLElement, rows: Record<string, unknown>[]): void {
  if (rows.length === 0) {
    el.innerHTML = '<p class="empty">no rows</p>';
    return;
  }
  const cols = Object.keys(rows[0]);
  const esc = (v: unknown) =>
    String(v ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;');
  el.innerHTML = `<table><thead><tr>${cols.map((c) => `<th>${esc(c)}</th>`).join('')}</tr></thead>
    <tbody>${rows
      .map((r) => `<tr>${cols.map((c) => `<td>${esc(fmt(r[c]))}</td>`).join('')}</tr>`)
      .join('')}</tbody></table>`;
}

function fmt(v: unknown): unknown {
  if (typeof v === 'number' && !Number.isInteger(v)) return v.toFixed(3);
  if (typeof v === 'bigint') return v.toString();
  return v;
}

/** series: {label, points: [epochSec, value][]}[] を1チャートに重ねる */
export function renderLine(
  el: HTMLElement,
  series: { label: string; points: [number, number][] }[],
): void {
  el.innerHTML = '';
  const xs = [...new Set(series.flatMap((s) => s.points.map((p) => p[0])))].sort((a, b) => a - b);
  const data: uPlot.AlignedData = [
    xs,
    ...series.map((s) => {
      const m = new Map(s.points);
      return xs.map((x) => m.get(x) ?? null);
    }),
  ];
  new uPlot(
    {
      width: Math.min(el.clientWidth || 900, 1200),
      height: 320,
      series: [{}, ...series.map((s, i) => ({ label: s.label, stroke: PALETTE[i % PALETTE.length] }))],
    },
    data,
    el,
  );
}

const PALETTE = ['#4269d0', '#efb118', '#ff725c', '#6cc5b0', '#3ca951', '#a463f2'];
