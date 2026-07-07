package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"

	"github.com/kyosu-1/testlake/collector/internal/store"
)

func TestRunCollect(t *testing.T) {
	dir := t.TempDir()
	// フィクスチャを流用
	src, _ := os.ReadFile("../../internal/junit/testdata/pytest.xml")
	reportDir := filepath.Join(dir, "reports")
	os.MkdirAll(reportDir, 0o755)
	os.WriteFile(filepath.Join(reportDir, "r.xml"), src, 0o644)

	dataDir := filepath.Join(dir, "staging")
	env := map[string]string{
		"GITHUB_RUN_ID": "7", "GITHUB_RUN_ATTEMPT": "1",
		"GITHUB_WORKFLOW": "CI", "GITHUB_JOB": "test",
		"GITHUB_REF_NAME": "main", "GITHUB_SHA": "deadbeef",
		"GITHUB_EVENT_NAME": "push", "RUNNER_OS": "Linux",
		"TESTLAKE_JOB_STATUS": "failure",
		"TESTLAKE_NOW":        "2026-07-08T10:00:00Z",
	}
	var out bytes.Buffer
	err := runCollect(
		[]string{"--reports", filepath.Join(reportDir, "**/*.xml"), "--data", dataDir},
		func(k string) string { return env[k] }, &out)
	if err != nil {
		t.Fatal(err)
	}
	tests, err := parquet.ReadFile[store.TestRow](
		filepath.Join(dataDir, "tests", "date=2026-07-08", "7-test-1.parquet"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tests) != 3 {
		t.Fatalf("want 3 test rows, got %d", len(tests))
	}
}

func TestRunCollectNoReportsIsWarningNotError(t *testing.T) {
	dir := t.TempDir()
	env := map[string]string{
		"GITHUB_RUN_ID": "7", "GITHUB_RUN_ATTEMPT": "1", "GITHUB_JOB": "test",
		"TESTLAKE_NOW": "2026-07-08T10:00:00Z",
	}
	var out bytes.Buffer
	err := runCollect(
		[]string{"--reports", filepath.Join(dir, "none/**/*.xml"), "--data", filepath.Join(dir, "s")},
		func(k string) string { return env[k] }, &out)
	if err != nil {
		t.Fatal(err) // レポートゼロでもエラーにしない(runs 行は書く)
	}
	if !strings.Contains(out.String(), "::warning::") {
		t.Errorf("want warning, got %q", out.String())
	}
}
