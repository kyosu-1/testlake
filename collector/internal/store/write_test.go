package store

import (
	"strings"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"

	"github.com/kyosu-1/testlake/collector/internal/ghctx"
	"github.com/kyosu-1/testlake/collector/internal/junit"
)

func meta() ghctx.RunMeta {
	return ghctx.RunMeta{
		RunID: 42, RunAttempt: 1, Workflow: "CI", Job: "unit tests",
		Branch: "main", SHA: "abc123", Event: "push", RunnerOS: "Linux",
		StartedAt:  time.Date(2026, 7, 8, 9, 5, 0, 0, time.UTC),
		DurationMS: 65000, Conclusion: "success",
	}
}

func TestWriteRunRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cases := []junit.TestCase{
		{Suite: "s", Class: "c", Name: "t1", Outcome: "passed", DurationMS: 10},
		{Suite: "s", Class: "c", Name: "t2", Outcome: "failed", DurationMS: 20, FailureMessage: "boom"},
	}
	runPath, testsPath, err := WriteRun(dir, meta(), cases)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(runPath, "runs/date=2026-07-08/42-unit-tests-1.parquet") {
		t.Errorf("runPath = %s", runPath)
	}
	runs, err := parquet.ReadFile[RunRow](runPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].SHA != "abc123" || runs[0].DurationMS != 65000 || runs[0].SchemaVersion != 1 {
		t.Errorf("runs = %+v", runs)
	}
	tests, err := parquet.ReadFile[TestRow](testsPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(tests) != 2 || tests[1].Outcome != "failed" || tests[1].SHA != "abc123" {
		t.Errorf("tests = %+v", tests)
	}
	if tests[0].StartedAt != meta().StartedAt.UnixMilli() {
		t.Errorf("StartedAt = %d", tests[0].StartedAt)
	}
}

func TestJobSlug(t *testing.T) {
	if got := JobSlug("unit tests (ubuntu / go1.22)"); got != "unit-tests--ubuntu---go1.22-" {
		t.Errorf("slug = %q", got)
	}
}
