package main

import (
	"regexp"
	"strings"
)

// Removed is the whole text of a rule that no longer holds; its ID stays listed.
const Removed = "Removed."

// OmitSameDayChanged is the owner's choice for a rule changed on the day it was added:
// true writes and expects the added date only; false writes and expects `changed` on that same day too.
const OmitSameDayChanged = true

// idShape is a rule ID as the constitutions write them: G-CON-1, V-7, H-2, P2-1.
const idShape = `[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)*-[0-9]+`

var (
	ruleLine = regexp.MustCompile(`^- \*\*(` + idShape + `)\*\*(?: (.*))?$`)
	// lookalike is any list item opening with a bold ID; one that is not a ruleLine is reported, never skipped.
	lookalike = regexp.MustCompile(`^\s*(?:[-*+]|[0-9]+[.)])\s+\*\*(` + idShape + `)\*\*`)
	datesPre  = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2})(?: · changed ([0-9]{4}-[0-9]{2}-[0-9]{2}))? · (.*)$`)
)

// Rule is one `- **ID** ...` bullet of a constitution file, with its indented lines.
type Rule struct {
	ID      string
	Added   string
	Changed string
	Text    string // the words after the dates, continuation lines included
	Line    int    // index of the bullet line in the file
	// Override is a project's line in its Overrides section, changing an org rule by its ID.
	Override bool
}

// NotRule is a list item that opens with a bold ID but is not written as a rule.
type NotRule struct {
	ID   string
	Line int
}

// Doc is a constitution file split into lines, with its rules found.
type Doc struct {
	Lines    []string
	Rules    []Rule
	NotRules []NotRule
}

// Parse finds the rules in a Markdown file; rules inside fenced code are examples and skipped.
func Parse(content string) Doc {
	d := Doc{Lines: strings.Split(content, "\n")}
	fenced := fencedLines(d.Lines)
	overrides := false
	for i := 0; i < len(d.Lines); i++ {
		if fenced[i] {
			continue
		}
		line := strings.TrimSuffix(d.Lines[i], "\r")
		if strings.HasPrefix(line, "#") {
			overrides = strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.TrimLeft(line, "#"))), "overrides")
			continue
		}
		m := ruleLine.FindStringSubmatch(line)
		if m == nil {
			d.noteNotRule(i)
			continue
		}
		r := Rule{ID: m[1], Line: i, Text: m[2], Override: overrides}
		if dm := datesPre.FindStringSubmatch(m[2]); dm != nil {
			r.Added, r.Changed, r.Text = dm[1], dm[2], dm[3]
		}
		end := i + 1
		for j := i + 1; j < len(d.Lines); j++ {
			next := d.Lines[j]
			if strings.TrimSpace(next) == "" {
				continue
			}
			if !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t") {
				break
			}
			end = j + 1
		}
		for j := i + 1; j < end; j++ {
			if !fenced[j] {
				d.noteNotRule(j)
			}
			r.Text += "\n" + d.Lines[j]
		}
		r.Text = normalize(r.Text)
		d.Rules = append(d.Rules, r)
		i = end - 1
	}
	return d
}

// noteNotRule records line i when it opens with a bold ID without being a rule.
func (d *Doc) noteNotRule(i int) {
	if m := lookalike.FindStringSubmatch(d.Lines[i]); m != nil {
		d.NotRules = append(d.NotRules, NotRule{ID: m[1], Line: i})
	}
}

// fencedLines marks the lines of fenced code, ``` or ~~~, fences included.
func fencedLines(lines []string) []bool {
	in := make([]bool, len(lines))
	fence := ""
	for i, l := range lines {
		t := strings.TrimSpace(l)
		switch {
		case fence != "":
			in[i] = true
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
		case strings.HasPrefix(t, "```"), strings.HasPrefix(t, "~~~"):
			in[i] = true
			fence = t[:3]
		}
	}
	return in
}

// normalize drops trailing spaces and carriage returns so a stray blank never counts as a change.
// Anything else does: a space inside a line, or the same words wrapped differently, is a text change.
func normalize(text string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t\r")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// changedDate is the changed date r stands for: none when it only repeats the added date and that is left out.
func changedDate(r Rule) string {
	if OmitSameDayChanged && r.Changed == r.Added {
		return ""
	}
	return r.Changed
}

// DatedLine is the bullet line of r carrying the given dates; changed is left out when empty,
// and when it is the added day and OmitSameDayChanged holds.
func DatedLine(r Rule, firstLine, added, changed string) string {
	eol := ""
	if strings.HasSuffix(firstLine, "\r") {
		firstLine, eol = strings.TrimSuffix(firstLine, "\r"), "\r"
	}
	_, words, _ := strings.Cut(firstLine, "** ")
	if dm := datesPre.FindStringSubmatch(words); dm != nil {
		words = dm[3]
	}
	dates := added
	if changed = changedDate(Rule{Added: added, Changed: changed}); changed != "" {
		dates += " · changed " + changed
	}
	return "- **" + r.ID + "** " + dates + " · " + words + eol
}
