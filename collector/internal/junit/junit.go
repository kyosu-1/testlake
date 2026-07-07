// Package junit parses JUnit-style XML reports leniently: any file or
// element it cannot understand is skipped, never fatal.
package junit

import (
	"encoding/xml"
	"os"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const maxFailureBytes = 4096

type TestCase struct {
	Suite          string
	Class          string
	Name           string
	File           string
	Outcome        string // passed | failed | error | skipped
	DurationMS     int64
	FailureMessage string
	FailureType    string
}

type xmlFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type xmlTestCase struct {
	Name      string      `xml:"name,attr"`
	ClassName string      `xml:"classname,attr"`
	File      string      `xml:"file,attr"`
	Time      string      `xml:"time,attr"`
	Failure   *xmlFailure `xml:"failure"`
	Error     *xmlFailure `xml:"error"`
	Skipped   *struct{}   `xml:"skipped"`
}

type xmlTestSuite struct {
	Name   string         `xml:"name,attr"`
	Cases  []xmlTestCase  `xml:"testcase"`
	Suites []xmlTestSuite `xml:"testsuite"` // Gradle 等のネスト対応
}

type xmlTestSuites struct {
	XMLName xml.Name       `xml:"testsuites"`
	Suites  []xmlTestSuite `xml:"testsuite"`
}

func ParseFile(path string) ([]TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root xmlTestSuites
	if err := xml.Unmarshal(data, &root); err == nil {
		var out []TestCase
		for _, s := range root.Suites {
			out = append(out, flatten(s)...)
		}
		return out, nil
	}
	// ルートが <testsuite> 単体のケース (pytest 等)
	var single xmlTestSuite
	if err := xml.Unmarshal(data, &single); err != nil {
		return nil, err
	}
	return flatten(single), nil
}

func flatten(s xmlTestSuite) []TestCase {
	var out []TestCase
	for _, c := range s.Cases {
		tc := TestCase{
			Suite:      s.Name,
			Class:      c.ClassName,
			Name:       c.Name,
			File:       c.File,
			Outcome:    "passed",
			DurationMS: parseSeconds(c.Time),
		}
		switch {
		case c.Failure != nil:
			tc.Outcome = "failed"
			tc.FailureMessage = truncate(firstNonEmpty(c.Failure.Message, c.Failure.Body), maxFailureBytes)
			tc.FailureType = c.Failure.Type
		case c.Error != nil:
			tc.Outcome = "error"
			tc.FailureMessage = truncate(firstNonEmpty(c.Error.Message, c.Error.Body), maxFailureBytes)
			tc.FailureType = c.Error.Type
		case c.Skipped != nil:
			tc.Outcome = "skipped"
		}
		out = append(out, tc)
	}
	for _, child := range s.Suites {
		out = append(out, flatten(child)...)
	}
	return out
}

// ParseGlobs parses every file matching the doublestar patterns.
// Unparseable files become warnings, never errors.
func ParseGlobs(patterns []string) ([]TestCase, []error) {
	var cases []TestCase
	var warns []error
	for _, p := range patterns {
		base, pat := doublestar.SplitPattern(strings.TrimSpace(p))
		matches, err := doublestar.Glob(os.DirFS(base), pat)
		if err != nil {
			warns = append(warns, err)
			continue
		}
		for _, m := range matches {
			cs, err := ParseFile(base + "/" + m)
			if err != nil {
				warns = append(warns, err)
				continue
			}
			cases = append(cases, cs...)
		}
	}
	return cases, warns
}

func parseSeconds(s string) int64 {
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		return 0
	}
	return int64(f * 1000)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return strings.TrimSpace(b)
}
