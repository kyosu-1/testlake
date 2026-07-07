// Command testlake collects CI test telemetry into Parquet (collect) and
// maintains a ci-data directory (finalize). It must never fail the host CI:
// problems surface as ::warning:: lines and exit code 0.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kyosu-1/testlake/collector/internal/ghapi"
	"github.com/kyosu-1/testlake/collector/internal/ghctx"
	"github.com/kyosu-1/testlake/collector/internal/junit"
	"github.com/kyosu-1/testlake/collector/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: testlake <collect|finalize> [flags]")
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "collect":
		err = runCollect(os.Args[2:], os.Getenv, os.Stdout)
	case "finalize":
		err = runFinalize(os.Args[2:], os.Stdout) // Task 8 で実装
	default:
		err = fmt.Errorf("unknown subcommand %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCollect(args []string, getenv func(string) string, stdout io.Writer) error {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	reports := fs.String("reports", "", "comma-separated JUnit XML globs")
	data := fs.String("data", "", "staging output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *reports == "" || *data == "" {
		return fmt.Errorf("--reports and --data are required")
	}

	meta, err := ghctx.FromEnv(getenv)
	if err != nil {
		return err
	}

	// job の実開始時刻を GH API から補正(失敗しても warning のみ)
	if tok := getenv("GITHUB_TOKEN"); tok != "" && getenv("TESTLAKE_NOW") == "" {
		started, err := ghapi.JobStartedAt(
			getenv("GITHUB_API_URL"), tok, getenv("GITHUB_REPOSITORY"),
			meta.RunID, meta.RunAttempt, meta.Job)
		if err != nil {
			fmt.Fprintf(stdout, "::warning::testlake: job start time unavailable: %v\n", err)
		} else {
			meta.DurationMS = time.Now().UTC().Sub(started).Milliseconds()
			meta.StartedAt = started
		}
	}

	cases, warns := junit.ParseGlobs(strings.Split(*reports, ","))
	for _, w := range warns {
		fmt.Fprintf(stdout, "::warning::testlake: report skipped: %v\n", w)
	}
	if len(cases) == 0 {
		fmt.Fprintf(stdout, "::warning::testlake: no test cases found in %q\n", *reports)
	}

	runPath, testsPath, err := store.WriteRun(*data, meta, cases)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "testlake: wrote %s (%d cases) and %s\n", testsPath, len(cases), runPath)
	return nil
}

func runFinalize(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("finalize", flag.ContinueOnError)
	data := fs.String("data", "", "ci-data directory")
	retention := fs.Int("retention-days", 400, "delete data older than this")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *data == "" {
		return fmt.Errorf("--data is required")
	}
	now := time.Now().UTC()
	if err := store.ApplyRetention(*data, now.AddDate(0, 0, -*retention)); err != nil {
		return err
	}
	if err := store.Compact(*data, now.AddDate(0, 0, -7)); err != nil {
		return err
	}
	m, err := store.BuildManifest(*data)
	if err != nil {
		return err
	}
	if err := store.WriteManifest(*data, m); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "testlake: finalized %s (%d run files, %d test files)\n",
		*data, len(m.Tables["runs"].Files), len(m.Tables["tests"].Files))
	return nil
}
