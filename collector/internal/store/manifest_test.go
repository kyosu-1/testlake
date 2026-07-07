package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildManifest(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := WriteRun(dir, meta(), nil); err != nil { // meta() は write_test.go のヘルパ
		t.Fatal(err)
	}
	m, err := BuildManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.SchemaVersion != 1 {
		t.Errorf("schema_version = %d", m.SchemaVersion)
	}
	rf := m.Tables["runs"].Files
	if len(rf) != 1 || rf[0].Path != "runs/date=2026-07-08/42-unit-tests-1.parquet" {
		t.Errorf("runs files = %+v", rf)
	}
	if rf[0].Rows != 1 || rf[0].Bytes <= 0 {
		t.Errorf("entry = %+v", rf[0])
	}
	if len(m.Tables["tests"].Files) != 1 {
		t.Errorf("tests files = %+v", m.Tables["tests"].Files)
	}

	if err := WriteManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var round map[string]any
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatal(err)
	}
	if round["schema_version"].(float64) != 1 {
		t.Errorf("json = %s", raw)
	}
}
