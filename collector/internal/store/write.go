// Package store owns the on-disk layout: Parquet row schemas, file naming,
// manifest, compaction. The layout is a public contract (docs/schema.md).
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/parquet-go/parquet-go"

	"github.com/kyosu-1/testlake/collector/internal/ghctx"
	"github.com/kyosu-1/testlake/collector/internal/junit"
)

const SchemaVersion = 1

type RunRow struct {
	SchemaVersion int32  `parquet:"schema_version"`
	RunID         int64  `parquet:"run_id"`
	RunAttempt    int32  `parquet:"run_attempt"`
	Workflow      string `parquet:"workflow"`
	Job           string `parquet:"job"`
	Branch        string `parquet:"branch"`
	SHA           string `parquet:"sha"`
	Event         string `parquet:"event"`
	RunnerOS      string `parquet:"runner_os"`
	StartedAt     int64  `parquet:"started_at,timestamp(millisecond)"`
	DurationMS    int64  `parquet:"duration_ms"`
	Conclusion    string `parquet:"conclusion"`
}

type TestRow struct {
	SchemaVersion  int32  `parquet:"schema_version"`
	RunID          int64  `parquet:"run_id"`
	RunAttempt     int32  `parquet:"run_attempt"`
	Job            string `parquet:"job"`
	Branch         string `parquet:"branch"`
	SHA            string `parquet:"sha"`
	StartedAt      int64  `parquet:"started_at,timestamp(millisecond)"`
	Suite          string `parquet:"suite"`
	Class          string `parquet:"class"`
	Name           string `parquet:"name"`
	File           string `parquet:"file"`
	Outcome        string `parquet:"outcome"`
	DurationMS     int64  `parquet:"duration_ms"`
	FailureMessage string `parquet:"failure_message"`
	FailureType    string `parquet:"failure_type"`
}

var slugRe = regexp.MustCompile(`[^a-zA-Z0-9._]`)

func JobSlug(job string) string { return slugRe.ReplaceAllString(job, "-") }

func WriteRun(dataDir string, m ghctx.RunMeta, cases []junit.TestCase) (string, string, error) {
	date := m.StartedAt.UTC().Format("2006-01-02")
	base := fmt.Sprintf("%d-%s-%d.parquet", m.RunID, JobSlug(m.Job), m.RunAttempt)

	runRow := RunRow{
		SchemaVersion: SchemaVersion, RunID: m.RunID, RunAttempt: m.RunAttempt,
		Workflow: m.Workflow, Job: m.Job, Branch: m.Branch, SHA: m.SHA,
		Event: m.Event, RunnerOS: m.RunnerOS,
		StartedAt: m.StartedAt.UnixMilli(), DurationMS: m.DurationMS, Conclusion: m.Conclusion,
	}
	runPath := filepath.Join(dataDir, "runs", "date="+date, base)
	if err := writeFile(runPath, []RunRow{runRow}); err != nil {
		return "", "", err
	}

	testRows := make([]TestRow, 0, len(cases))
	for _, c := range cases {
		testRows = append(testRows, TestRow{
			SchemaVersion: SchemaVersion, RunID: m.RunID, RunAttempt: m.RunAttempt,
			Job: m.Job, Branch: m.Branch, SHA: m.SHA, StartedAt: m.StartedAt.UnixMilli(),
			Suite: c.Suite, Class: c.Class, Name: c.Name, File: c.File,
			Outcome: c.Outcome, DurationMS: c.DurationMS,
			FailureMessage: c.FailureMessage, FailureType: c.FailureType,
		})
	}
	testsPath := filepath.Join(dataDir, "tests", "date="+date, base)
	if err := writeFile(testsPath, testRows); err != nil {
		return "", "", err
	}
	return runPath, testsPath, nil
}

func writeFile[T any](path string, rows []T) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return parquet.WriteFile(path, rows)
}
