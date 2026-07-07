// Package ghctx extracts run metadata from GitHub Actions environment variables.
package ghctx

import (
	"fmt"
	"strconv"
	"time"
)

type RunMeta struct {
	RunID      int64
	RunAttempt int32
	Workflow   string
	Job        string
	Branch     string
	SHA        string
	Event      string
	RunnerOS   string
	StartedAt  time.Time // job 開始時刻。ghapi で上書きされるまでは「現在時刻」
	DurationMS int64     // 0 = 不明
	Conclusion string
}

func FromEnv(getenv func(string) string) (RunMeta, error) {
	runID, err := strconv.ParseInt(getenv("GITHUB_RUN_ID"), 10, 64)
	if err != nil {
		return RunMeta{}, fmt.Errorf("GITHUB_RUN_ID: %w", err)
	}
	attempt, err := strconv.ParseInt(getenv("GITHUB_RUN_ATTEMPT"), 10, 32)
	if err != nil {
		attempt = 1
	}
	now := time.Now().UTC()
	if v := getenv("TESTLAKE_NOW"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			now = t.UTC()
		}
	}
	conclusion := getenv("TESTLAKE_JOB_STATUS")
	if conclusion == "" {
		conclusion = "unknown"
	}
	return RunMeta{
		RunID:      runID,
		RunAttempt: int32(attempt),
		Workflow:   getenv("GITHUB_WORKFLOW"),
		Job:        getenv("GITHUB_JOB"),
		Branch:     getenv("GITHUB_REF_NAME"),
		SHA:        getenv("GITHUB_SHA"),
		Event:      getenv("GITHUB_EVENT_NAME"),
		RunnerOS:   getenv("RUNNER_OS"),
		StartedAt:  now,
		Conclusion: conclusion,
	}, nil
}
