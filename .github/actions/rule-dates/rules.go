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

var (
	ruleLine = regexp.MustCompile(`^- \*\*([A-Z]+(?:-[A-Z]+)*-[0-9]+)\*\*(?: (.*))?$`)
	datesPre = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2})(?: · changed ([0-9]{4}-[0-9]{2}-[0-9]{2}))? · (.*)$`)
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

// Doc is a constitution file split into lines, with its rules found.
type Doc struct {
	Lines []string
	Rules []Rule
}

// Parse finds the rules in a Markdown file; rules inside fenced code are examples and skipped.
func Parse(content string) Doc {
	d := Doc{Lines: strings.Split(content, "\n")}
	fenced, overrides := false, false
	for i := 0; i < len(d.Lines); i++ {
		line := d.Lines[i]
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if !fenced && strings.HasPrefix(line, "#") {
			overrides = strings.EqualFold(strings.TrimSpace(strings.TrimLeft(line, "#")), "Overrides")
			continue
		}
		m := ruleLine.FindStringSubmatch(line)
		if fenced || m == nil {
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
		for _, c := range d.Lines[i+1 : end] {
			r.Text += "\n" + c
		}
		r.Text = normalize(r.Text)
		d.Rules = append(d.Rules, r)
		i = end - 1
	}
	return d
}

// normalize drops trailing spaces so a stray blank never counts as a change.
func normalize(text string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
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
	_, words, _ := strings.Cut(firstLine, "** ")
	if dm := datesPre.FindStringSubmatch(words); dm != nil {
		words = dm[3]
	}
	dates := added
	if changed = changedDate(Rule{Added: added, Changed: changed}); changed != "" {
		dates += " · changed " + changed
	}
	return "- **" + r.ID + "** " + dates + " · " + words
}
