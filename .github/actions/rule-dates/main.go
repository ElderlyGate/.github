// rule-dates checks that every constitution rule carries the day it was added and last changed (G-CON-8),
// and backfills those dates once from git history.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

const usage = `rule-dates: every constitution rule says when it was added and changed (G-CON-8)

Start:
  rule-dates <command> [flags] [pattern...]   run at the repository root; patterns default to constitution/*.md

Commands:
  check      fail when a rule has no date, or its text changed without its changed date
  backfill   work out every rule's dates from git and show the lines it would change

Flags:
  -base <rev>          check: compare with this revision (a pull request's base)
  -today <YYYY-MM-DD>  check: the day a change is dated (default: today, UTC)
  -rev <rev>           backfill: read history up to this revision (default: HEAD)
  -write               backfill: write the dates into the files

More: https://github.com/ElderlyGate/.github
`

func main() {
	os.Exit(run(os.Args[1:], ".", os.Stdout, os.Stderr))
}

// run is the whole command; it returns the exit code: 0 fine, 1 problems found, 2 could not run.
func run(args []string, dir string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	base := fs.String("base", "", "")
	today := fs.String("today", time.Now().UTC().Format(time.DateOnly), "")
	rev := fs.String("rev", "HEAD", "")
	write := fs.Bool("write", false, "")
	if err := fs.Parse(args[1:]); err != nil {
		fmt.Fprint(stderr, usage)
		return 2
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"constitution/*.md"}
	}
	var err error
	code := 0
	switch args[0] {
	case "check":
		code, err = check(dir, *base, *today, patterns, stdout)
	case "backfill":
		code, err = backfill(dir, *rev, *write, patterns, stdout)
	default:
		fmt.Fprintf(stderr, "rule-dates: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, "rule-dates:", err)
		return 2
	}
	return code
}

func check(dir, base, today string, patterns []string, out io.Writer) (int, error) {
	head, err := WorkFiles(dir, patterns)
	if err != nil {
		return 0, err
	}
	var before Files
	var baseDay string
	if base != "" {
		if before, err = RevFiles(dir, base, patterns); err != nil {
			return 0, err
		}
		if baseDay, err = CommitDay(dir, base); err != nil {
			return 0, err
		}
	}
	problems := Check(before, baseDay, head, today)
	for _, p := range problems {
		fmt.Fprintln(out, p)
	}
	if len(problems) > 0 {
		fmt.Fprintf(out, "%d problem(s); see G-CON-8\n", len(problems))
		return 1, nil
	}
	fmt.Fprintf(out, "rule dates fine in %d file(s)\n", len(head))
	return 0, nil
}

func backfill(dir, rev string, write bool, patterns []string, out io.Writer) (int, error) {
	snaps, err := Snapshots(dir, rev, patterns)
	if err != nil {
		return 0, err
	}
	files, err := WorkFiles(dir, patterns)
	if err != nil {
		return 0, err
	}
	dates, odd := Learn(snaps)
	dated, edits, unknown := Apply(files, dates)
	for _, e := range edits {
		fmt.Fprintf(out, "%s:%d\n- %s\n+ %s\n", e.Path, e.Line, e.Before, e.After)
	}
	for _, o := range append(odd, unknown...) {
		fmt.Fprintln(out, "odd:", o)
	}
	rules, _ := index(files)
	fmt.Fprintf(out, "%d rule(s) in %d file(s), %d line(s) to change, %d commit(s) read\n", len(rules), len(files), len(edits), len(snaps))
	if !write {
		return 0, nil
	}
	changed := Files{}
	for _, e := range edits {
		changed[e.Path] = dated[e.Path]
	}
	if err := Write(dir, changed); err != nil {
		return 0, err
	}
	return 0, nil
}
