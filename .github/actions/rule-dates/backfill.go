package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Dates is what history says about one rule.
type Dates struct {
	Added   string
	Changed string
}

// Snapshot is the constitution files as one commit left them.
type Snapshot struct {
	Date  string
	Files Files
}

// Edit is one bullet line the backfill rewrites.
type Edit struct {
	Path   string
	Line   int
	Before string
	After  string
}

// ruleState follows one ID through history.
type ruleState struct {
	Dates
	text    string
	present bool
}

// Snapshots reads the files at every first-parent commit up to rev that touched them, oldest first.
func Snapshots(dir, rev string, patterns []string) ([]Snapshot, error) {
	commits, err := History(dir, rev, patterns)
	if err != nil {
		return nil, err
	}
	snaps := make([]Snapshot, 0, len(commits))
	for _, c := range commits {
		files, err := RevFiles(dir, c.SHA, patterns)
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, Snapshot{Date: c.Date, Files: files})
	}
	return snaps, nil
}

// Learn works out each rule's dates from history, and notes what looks odd on the way.
// Added is the first snapshot holding the ID; changed is the last one that changed its text.
func Learn(snaps []Snapshot) (map[string]Dates, []string) {
	states := map[string]*ruleState{}
	var odd []string
	var prevPlain map[string]bool
	prevByText := map[string]string{}
	reported := map[string]bool{}
	for _, s := range snaps {
		now, twice := index(s.Files)
		for _, again := range twice {
			if !reported[again.ID] {
				reported[again.ID] = true
				odd = append(odd, fmt.Sprintf("%s: listed twice from %s; dates follow the first (%s)", again.ID, s.Date, usedTwice(now, again)))
			}
		}
		byText := map[string]string{}
		for _, id := range sortedIDs(now) {
			r := now[id]
			byText[r.Text] = id
			st, seen := states[id]
			switch {
			case !seen:
				states[id] = &ruleState{Dates: Dates{Added: s.Date}, text: r.Text, present: true}
				if other, ok := prevByText[r.Text]; ok && now[other].ID == "" {
					odd = append(odd, fmt.Sprintf("%s: added %s with the text of %s, which left that day", id, s.Date, other))
				} else if prevPlain[firstLine(r.Text)] {
					odd = append(odd, fmt.Sprintf("%s: added %s, but its text was there before, without an ID", id, s.Date))
				}
				continue
			case !st.present:
				odd = append(odd, fmt.Sprintf("%s: came back %s after it was gone", id, s.Date))
			}
			if r.Text != st.text {
				if st.text == Removed {
					odd = append(odd, fmt.Sprintf("%s: was removed, then reused %s", id, s.Date))
				}
				st.Changed = s.Date
				st.text = r.Text
			}
			st.present = true
		}
		for id, st := range states {
			if _, ok := now[id]; !ok {
				st.present = false
			}
		}
		prevPlain = plainLines(s.Files)
		prevByText = byText
	}
	dates := map[string]Dates{}
	for id, st := range states {
		dates[id] = st.Dates
	}
	return dates, odd
}

// Apply writes the dates into the bullet lines of files; it returns the edits, and the rules history does not know
// with the lines that look like rules but are not.
func Apply(files Files, dates map[string]Dates) (Files, []Edit, []string) {
	out := Files{}
	var edits []Edit
	var unknown []string
	for _, p := range sortedPaths(files) {
		doc := Parse(files[p])
		for _, n := range doc.NotRules {
			unknown = append(unknown, fmt.Sprintf("%s:%d: %s is in bold like a rule ID but is not read as a rule; left as it is", p, n.Line+1, n.ID))
		}
		for _, r := range doc.Rules {
			d, ok := dates[r.ID]
			if !ok {
				unknown = append(unknown, fmt.Sprintf("%s:%d: %s has no history; left without a date", p, r.Line+1, r.ID))
				continue
			}
			before := doc.Lines[r.Line]
			after := DatedLine(r, before, d.Added, d.Changed)
			if after != before {
				doc.Lines[r.Line] = after
				edits = append(edits, Edit{Path: p, Line: r.Line + 1, Before: before, After: after})
			}
		}
		out[p] = strings.Join(doc.Lines, "\n")
	}
	return out, edits, unknown
}

// Write puts each file in place whole: a temporary file beside it, then a rename (G-CODE-5).
func Write(dir string, files Files) error {
	for _, p := range sortedPaths(files) {
		target := filepath.Join(dir, filepath.FromSlash(p))
		info, err := os.Stat(target)
		if err != nil {
			return err
		}
		tmp, err := os.CreateTemp(filepath.Dir(target), ".rule-dates-*")
		if err != nil {
			return err
		}
		_, werr := tmp.WriteString(files[p])
		cerr := tmp.Close()
		if werr == nil {
			werr = cerr
		}
		if werr == nil {
			werr = os.Chmod(tmp.Name(), info.Mode())
		}
		if werr == nil {
			werr = os.Rename(tmp.Name(), target)
		}
		if werr != nil {
			os.Remove(tmp.Name())
			return werr
		}
	}
	return nil
}

// plainLines are the bullet and text lines of files that carry no rule ID, as a rule's first line would read.
func plainLines(files Files) map[string]bool {
	plain := map[string]bool{}
	for _, content := range files {
		for _, l := range strings.Split(content, "\n") {
			if ruleLine.MatchString(l) {
				continue
			}
			l = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- "))
			if l != "" {
				plain[l] = true
			}
		}
	}
	return plain
}

// firstLine is the first line of a rule's text.
func firstLine(text string) string {
	first, _, _ := strings.Cut(text, "\n")
	return first
}

// sortedIDs gives the IDs of rules in a stable order.
func sortedIDs(rules map[string]placed) []string {
	ids := make([]string, 0, len(rules))
	for id := range rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
