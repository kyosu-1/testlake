package junit

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParsePytest(t *testing.T) {
	cases, err := ParseFile("testdata/pytest.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 3 {
		t.Fatalf("want 3 cases, got %d", len(cases))
	}
	if cases[0].Outcome != "passed" || cases[0].DurationMS != 501 || cases[0].File != "tests/test_auth.py" {
		t.Errorf("case0 = %+v", cases[0])
	}
	if cases[1].Outcome != "failed" || cases[1].FailureMessage != "AssertionError: expected 200 got 500" {
		t.Errorf("case1 = %+v", cases[1])
	}
	if cases[2].Outcome != "skipped" {
		t.Errorf("case2 = %+v", cases[2])
	}
	if cases[0].Suite != "pytest" || cases[0].Class != "tests.test_auth" {
		t.Errorf("suite/class = %q/%q", cases[0].Suite, cases[0].Class)
	}
}

func TestParseJestErrorOutcome(t *testing.T) {
	cases, err := ParseFile("testdata/jest.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 2 {
		t.Fatalf("want 2, got %d", len(cases))
	}
	if cases[1].Outcome != "error" || cases[1].FailureType != "TypeError" {
		t.Errorf("case1 = %+v", cases[1])
	}
	if cases[0].Suite != "Login.test.ts" {
		t.Errorf("suite = %q", cases[0].Suite)
	}
}

func TestTruncateFailureMessage(t *testing.T) {
	long := strings.Repeat("x", 10000)
	if got := truncate(long, 4096); len(got) != 4096 {
		t.Errorf("len = %d", len(got))
	}
}

func TestTruncateUTF8Safe(t *testing.T) {
	// "あ" is a 3-byte rune in UTF-8; repeating it ensures the naive byte
	// cut at n=4096 lands mid-rune (4096 is not a multiple of 3).
	long := strings.Repeat("あ", 2000) // 6000 bytes
	got := truncate(long, 4096)
	if !utf8.ValidString(got) {
		t.Fatalf("truncate produced invalid UTF-8: %q", got)
	}
	if len(got) > 4096 {
		t.Fatalf("len = %d, want <= 4096", len(got))
	}
}

func TestParseGlobsLenient(t *testing.T) {
	pattern := filepath.Join("testdata", "*.xml") // broken.xml を含む
	cases, warns := ParseGlobs([]string{pattern})
	if len(cases) != 6 { // pytest 3 + jest 2 + gotestsum 1
		t.Fatalf("want 6 cases, got %d", len(cases))
	}
	if len(warns) != 1 {
		t.Fatalf("want 1 warning for broken.xml, got %d: %v", len(warns), warns)
	}
}
