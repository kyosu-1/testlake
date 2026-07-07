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

func compactTable[T any](tableDir string, olderThan time.Time) error {
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
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) <= 1 {
			continue
		}
		var all []T
		var paths []string
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".parquet") {
				continue
			}
			p := filepath.Join(dir, e.Name())
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
		tmp := filepath.Join(dir, ".compacting.parquet")
		if err := parquet.WriteFile(tmp, all); err != nil {
			return err
		}
		for _, p := range paths {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
		if err := os.Rename(tmp, filepath.Join(dir, "compacted.parquet")); err != nil {
			return err
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
