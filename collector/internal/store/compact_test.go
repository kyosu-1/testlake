package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"
)

func writeAt(t *testing.T, dir string, day string, runID int64) {
	t.Helper()
	m := meta()
	m.RunID = runID
	m.StartedAt, _ = time.Parse(time.RFC3339, day+"T09:00:00Z")
	if _, _, err := WriteRun(dir, m, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCompactMergesOldDays(t *testing.T) {
	dir := t.TempDir()
	writeAt(t, dir, "2026-06-01", 1)
	writeAt(t, dir, "2026-06-01", 2)
	writeAt(t, dir, "2026-07-08", 3) // 新しい日 → 触らない

	cutoff, _ := time.Parse("2006-01-02", "2026-07-01")
	if err := Compact(dir, cutoff); err != nil {
		t.Fatal(err)
	}
	oldDir := filepath.Join(dir, "runs", "date=2026-06-01")
	entries, _ := os.ReadDir(oldDir)
	if len(entries) != 1 || entries[0].Name() != "compacted.parquet" {
		t.Fatalf("entries = %v", entries)
	}
	rows, err := parquet.ReadFile[RunRow](filepath.Join(oldDir, "compacted.parquet"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 merged rows, got %d", len(rows))
	}
	newDir := filepath.Join(dir, "runs", "date=2026-07-08")
	entries, _ = os.ReadDir(newDir)
	if len(entries) != 1 || entries[0].Name() == "compacted.parquet" {
		t.Fatalf("new day should be untouched: %v", entries)
	}
}

func TestCompactSkipsStaleTmpFile(t *testing.T) {
	dir := t.TempDir()
	writeAt(t, dir, "2026-06-01", 1)
	writeAt(t, dir, "2026-06-01", 2)

	oldDir := filepath.Join(dir, "runs", "date=2026-06-01")
	entries, err := os.ReadDir(oldDir)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a crashed previous compaction run: a stale .compacting.parquet
	// left behind, containing one source file's rows.
	srcPath := filepath.Join(oldDir, entries[0].Name())
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, ".compacting.parquet"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cutoff, _ := time.Parse("2006-01-02", "2026-07-01")
	if err := Compact(dir, cutoff); err != nil {
		t.Fatal(err)
	}

	entries, _ = os.ReadDir(oldDir)
	if len(entries) != 1 || entries[0].Name() != "compacted.parquet" {
		t.Fatalf("entries = %v", entries)
	}
	rows, err := parquet.ReadFile[RunRow](filepath.Join(oldDir, "compacted.parquet"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows (stale tmp must not be treated as an input), got %d", len(rows))
	}
}

func TestCompactDedupsIdenticalRows(t *testing.T) {
	dir := t.TempDir()
	writeAt(t, dir, "2026-06-01", 1)

	oldDir := filepath.Join(dir, "runs", "date=2026-06-01")
	entries, err := os.ReadDir(oldDir)
	if err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(oldDir, entries[0].Name())
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	// Same rows show up under a second file name (e.g. leftover from a
	// partially-failed compaction that renamed but couldn't delete sources).
	if err := os.WriteFile(filepath.Join(oldDir, "dup-copy.parquet"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cutoff, _ := time.Parse("2006-01-02", "2026-07-01")
	if err := Compact(dir, cutoff); err != nil {
		t.Fatal(err)
	}

	entries, _ = os.ReadDir(oldDir)
	if len(entries) != 1 || entries[0].Name() != "compacted.parquet" {
		t.Fatalf("entries = %v", entries)
	}
	rows, err := parquet.ReadFile[RunRow](filepath.Join(oldDir, "compacted.parquet"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 deduped row, got %d", len(rows))
	}
}

func TestApplyRetention(t *testing.T) {
	dir := t.TempDir()
	writeAt(t, dir, "2025-01-01", 1)
	writeAt(t, dir, "2026-07-08", 2)
	cutoff, _ := time.Parse("2006-01-02", "2026-06-01")
	if err := ApplyRetention(dir, cutoff); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "runs", "date=2025-01-01")); !os.IsNotExist(err) {
		t.Error("old dir should be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "runs", "date=2026-07-08")); err != nil {
		t.Error("recent dir should remain")
	}
}
