package ghctx

import (
	"testing"
	"time"
)

func stubEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestFromEnv(t *testing.T) {
	env := stubEnv(map[string]string{
		"GITHUB_RUN_ID":      "9876543210",
		"GITHUB_RUN_ATTEMPT": "2",
		"GITHUB_WORKFLOW":    "CI",
		"GITHUB_JOB":         "unit-tests",
		"GITHUB_REF_NAME":    "main",
		"GITHUB_SHA":         "abc123",
		"GITHUB_EVENT_NAME":  "push",
		"RUNNER_OS":          "Linux",
		"TESTLAKE_JOB_STATUS": "success",
		"TESTLAKE_NOW":       "2026-07-08T10:00:00Z",
	})
	m, err := FromEnv(env)
	if err != nil {
		t.Fatal(err)
	}
	if m.RunID != 9876543210 || m.RunAttempt != 2 || m.Job != "unit-tests" {
		t.Errorf("meta = %+v", m)
	}
	if m.Conclusion != "success" || m.Branch != "main" {
		t.Errorf("meta = %+v", m)
	}
	want := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)
	if !m.StartedAt.Equal(want) {
		t.Errorf("StartedAt = %v, want %v", m.StartedAt, want)
	}
}

func TestFromEnvMissingRunID(t *testing.T) {
	if _, err := FromEnv(stubEnv(map[string]string{})); err == nil {
		t.Fatal("want error when GITHUB_RUN_ID missing")
	}
}
