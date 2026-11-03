package main

import "testing"

func TestParseReadsDatesTextAndIndentedLines(t *testing.T) {
	doc := Parse(`# Tech stack

- **G-CON-1** 2026-10-27 · changed 2026-10-28 · The constitution is the source of truth.
- **G-CON-7** 2026-10-27 · Found in this order:
  1. hubman.

  If no step works, STOP.
- **H-1** No dates yet.

Some text.
`)
	if len(doc.Rules) != 3 {
		t.Fatalf("rules = %d, want 3", len(doc.Rules))
	}
	first := doc.Rules[0]
	if first.ID != "G-CON-1" || first.Added != "2026-10-27" || first.Changed != "2026-10-28" || first.Text != "The constitution is the source of truth." {
		t.Errorf("first rule = %+v", first)
	}
	if got := doc.Rules[1].Text; got != "Found in this order:\n  1. hubman.\n\n  If no step works, STOP." {
		t.Errorf("indented lines = %q", got)
	}
	if r := doc.Rules[2]; r.Added != "" || r.Changed != "" || r.Text != "No dates yet." {
		t.Errorf("undated rule = %+v", r)
	}
}

func TestParseSkipsRulesInFencedCode(t *testing.T) {
	doc := Parse("```\n- **G-STACK-5** An example.\n```\n- **G-STACK-6** Real.\n")
	if len(doc.Rules) != 1 || doc.Rules[0].ID != "G-STACK-6" {
		t.Errorf("rules = %+v, want only G-STACK-6", doc.Rules)
	}
}

func TestDatedLineReplacesOldDatesAndLeavesChangedOutWhenEmpty(t *testing.T) {
	r := Rule{ID: "V-1"}
	if got := DatedLine(r, "- **V-1** A version is a tag.", "2026-10-27", ""); got != "- **V-1** 2026-10-27 · A version is a tag." {
		t.Errorf("added only = %q", got)
	}
	if got := DatedLine(r, "- **V-1** 2026-01-01 · A version is a tag.", "2026-10-27", "2026-10-28"); got != "- **V-1** 2026-10-27 · changed 2026-10-28 · A version is a tag." {
		t.Errorf("both dates = %q", got)
	}
}
