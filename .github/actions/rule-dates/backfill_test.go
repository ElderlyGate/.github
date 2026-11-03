package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func snap(date string, lines ...string) Snapshot {
	return Snapshot{Date: date, Files: file(lines...)}
}

func TestLearnTakesTheFirstDayOfAnIDAndTheLastChangeOfItsText(t *testing.T) {
	dates, odd := Learn([]Snapshot{
		snap("2026-10-27", "- **G-CON-1** First words."),
		snap("2026-10-28", "- **G-CON-1** First words.", "- **G-CON-2** Two."),
		snap("2026-10-29", "- **G-CON-1** Better words.", "- **G-CON-2** Two."),
		snap("2026-10-30", "- **G-CON-1** Better words.", "- **G-CON-2** Two.", "Some prose."),
	})
	if got := dates["G-CON-1"]; got != (Dates{Added: "2026-10-27", Changed: "2026-10-29"}) {
		t.Errorf("G-CON-1 = %+v", got)
	}
	if got := dates["G-CON-2"]; got != (Dates{Added: "2026-10-28"}) {
		t.Errorf("G-CON-2 = %+v", got)
	}
	expectProblems(t, odd)
}

func TestLearnNotesWhatLooksOdd(t *testing.T) {
	dates, odd := Learn([]Snapshot{
		snap("2026-10-27", "- A rule without an ID.", "- **V-1** Words.", "- **V-2** Old text."),
		snap("2026-10-28", "- **G-1** A rule without an ID.", "- **V-1** Removed.", "- **V-3** Old text."),
		snap("2026-10-29", "- **G-1** A rule without an ID.", "- **V-1** Back again."),
		snap("2026-10-30", "- **G-1** A rule without an ID.", "- **V-1** Back again.", "- **V-3** Old text."),
	})
	expectProblems(t, odd,
		"G-1: added 2026-10-28, but its text was there before, without an ID",
		"V-3: added 2026-10-28 with the text of V-2, which left that day",
		"V-1: was removed, then reused 2026-10-29",
		"V-3: came back 2026-10-30 after it was gone",
	)
	if got := dates["V-1"]; got != (Dates{Added: "2026-10-27", Changed: "2026-10-29"}) {
		t.Errorf("V-1 = %+v", got)
	}
}

func TestApplyDatesKnownRulesAndReportsTheRest(t *testing.T) {
	files := file("# Rules", "- **V-1** Words.", "- **V-9** New.")
	out, edits, unknown := Apply(files, map[string]Dates{"V-1": {Added: "2026-10-27", Changed: "2026-10-28"}})
	if got := out["constitution/tech-stack.md"]; got != "# Rules\n- **V-1** 2026-10-27 · changed 2026-10-28 · Words.\n- **V-9** New.\n" {
		t.Errorf("file = %q", got)
	}
	if len(edits) != 1 || edits[0].Line != 2 {
		t.Errorf("edits = %+v", edits)
	}
	expectProblems(t, unknown, "constitution/tech-stack.md:3: V-9 has no history")
}

// TestBackfillThenCheckOnARealRepository runs both commands the way CI and the owner do.
func TestBackfillThenCheckOnARealRepository(t *testing.T) {
	dir := t.TempDir()
	gitIn := func(date string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "HOME="+dir, "XDG_CONFIG_HOME="+dir, "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_DATE="+date+"T12:00:00Z", "GIT_COMMITTER_DATE="+date+"T12:00:00Z")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit := func(date, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "constitution", "tech-stack.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		gitIn(date, "add", "-A")
		gitIn(date, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "docs: rules")
	}
	gitIn("2026-10-27", "init", "-q", "-b", "main")
	if err := os.MkdirAll(filepath.Join(dir, "constitution"), 0o755); err != nil {
		t.Fatal(err)
	}
	commit("2026-10-27", "- **P-1** One.\n")
	commit("2026-10-29", "- **P-1** One, better.\n- **P-2** Two.\n")

	var out bytes.Buffer
	if code := run([]string{"backfill", "-write"}, dir, &out, &out); code != 0 {
		t.Fatalf("backfill exit %d: %s", code, out.String())
	}
	b, _ := os.ReadFile(filepath.Join(dir, "constitution", "tech-stack.md"))
	if want := "- **P-1** 2026-10-27 · changed 2026-10-29 · One, better.\n- **P-2** 2026-10-29 · Two.\n"; string(b) != want {
		t.Fatalf("after backfill = %q, want %q", b, want)
	}

	out.Reset()
	if code := run([]string{"check", "-base", "HEAD", "-today", "2026-11-03"}, dir, &out, &out); code != 0 {
		t.Fatalf("check after backfill exit %d: %s", code, out.String())
	}
	if err := os.WriteFile(filepath.Join(dir, "constitution", "tech-stack.md"), []byte(strings.Replace(string(b), "Two.", "Two, new.", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run([]string{"check", "-base", "HEAD", "-today", "2026-11-03"}, dir, &out, &out); code != 1 || !strings.Contains(out.String(), "P-2 changed its text") {
		t.Fatalf("check of an undated change exit %d: %s", code, out.String())
	}
}
