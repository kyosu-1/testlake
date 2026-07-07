package store

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"
)

func Compact(dataDir string, olderThan time.Time) error {
	if err := compactTable[RunRow](filepath.Join(dataDir, "runs"), olderThan); err != nil {
		return err
	}
	return compactTable[TestRow](filepath.Join(dataDir, "tests"), olderThan)
}

func compactTable[T comparable](tableDir string, olderThan time.Time) error {
	days, err := os.ReadDir(tableDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, day := range days {
		date, ok := parseDateDir(day)
		if !ok || !date.Before(olderThan) {
			continue
		}
		dir := filepath.Join(tableDir, day.Name())

		tmp := filepath.Join(dir, ".compacting.parquet")
		// A stale tmp file left behind by a crashed/interrupted previous run
		// must be removed before we start: it is never a valid input (see
		// filter below), and it must not be left dangling if this run
		// doesn't otherwise touch the directory.
		if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) {
			return err
		}

		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) <= 1 {
			continue
		}
		var all []T
		var paths []string
		for _, e := range entries {
			name := e.Name()
			// Dot-prefixed files (in particular a stale .compacting.parquet)
			// are never valid compaction inputs.
			if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".parquet") {
				continue
			}
			p := filepath.Join(dir, name)
			rows, err := parquet.ReadFile[T](p)
			if err != nil {
				continue // 読めないファイルは残す(寛容)
			}
			all = append(all, rows...)
			paths = append(paths, p)
		}
		if len(paths) <= 1 {
			continue
		}

		// Dedup identical rows while merging: if a previous compaction
		// crashed after the rename but before deleting sources,
		// compacted.parquet and a stale source can both be read as inputs
		// here, and would otherwise be duplicated forever. Preserve
		// first-seen order.
		seen := make(map[T]struct{}, len(all))
		deduped := make([]T, 0, len(all))
		for _, row := range all {
			if _, ok := seen[row]; ok {
				continue
			}
			seen[row] = struct{}{}
			deduped = append(deduped, row)
		}

		if err := parquet.WriteFile(tmp, deduped); err != nil {
			return err
		}
		compactedPath := filepath.Join(dir, "compacted.parquet")
		// Commit point: once this succeeds, the merged data is durable
		// under its final name. Only after this may we delete sources.
		if err := os.Rename(tmp, compactedPath); err != nil {
			return err
		}
		for _, p := range paths {
			// compacted.parquet can itself be one of the read inputs when
			// re-compacting a day that gained new files since the last
			// compaction; it is also the rename target above, so it must
			// never appear in the delete list.
			if p == compactedPath {
				continue
			}
			if err := os.Remove(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func ApplyRetention(dataDir string, cutoff time.Time) error {
	for _, table := range []string{"runs", "tests"} {
		tableDir := filepath.Join(dataDir, table)
		days, err := os.ReadDir(tableDir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		for _, day := range days {
			if date, ok := parseDateDir(day); ok && date.Before(cutoff) {
				if err := os.RemoveAll(filepath.Join(tableDir, day.Name())); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func parseDateDir(d os.DirEntry) (time.Time, bool) {
	if !d.IsDir() || !strings.HasPrefix(d.Name(), "date=") {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", strings.TrimPrefix(d.Name(), "date="))
	return t, err == nil
}
