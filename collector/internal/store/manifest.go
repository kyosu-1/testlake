package store

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"
)

type FileEntry struct {
	Path  string `json:"path"`
	Rows  int64  `json:"rows"`
	Bytes int64  `json:"bytes"`
}

type TableManifest struct {
	Files []FileEntry `json:"files"`
}

type Manifest struct {
	SchemaVersion int                      `json:"schema_version"`
	GeneratedAt   time.Time                `json:"generated_at"`
	Tables        map[string]TableManifest `json:"tables"`
}

func BuildManifest(dataDir string) (Manifest, error) {
	m := Manifest{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		Tables:        map[string]TableManifest{},
	}
	for _, table := range []string{"runs", "tests"} {
		root := filepath.Join(dataDir, table)
		var files []FileEntry
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".parquet") {
				return nil //nolint:nilerr // 読めないエントリはスキップ(寛容)
			}
			st, err := os.Stat(path)
			if err != nil {
				return nil
			}
			rows := int64(-1)
			if f, err := os.Open(path); err == nil {
				if pf, err := parquet.OpenFile(f, st.Size()); err == nil {
					rows = pf.NumRows()
				}
				f.Close()
			}
			rel, _ := filepath.Rel(dataDir, path)
			files = append(files, FileEntry{Path: filepath.ToSlash(rel), Rows: rows, Bytes: st.Size()})
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return m, err
		}
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
		m.Tables[table] = TableManifest{Files: files}
	}
	return m, nil
}

func WriteManifest(dataDir string, m Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, "manifest.json"), data, 0o644)
}
