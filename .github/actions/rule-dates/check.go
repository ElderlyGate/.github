package main

import (
	"fmt"
	"time"
)

// placed is a rule and the file it was found in.
type placed struct {
	Rule
	Path string
}

// index finds every rule in files by ID; it keeps the first of an ID found twice and returns the others.
func index(files Files) (map[string]placed, []placed) {
	rules := map[string]placed{}
	var twice []placed
	for _, p := range sortedPaths(files) {
		for _, r := range Parse(files[p]).Rules {
			if _, ok := rules[r.ID]; ok {
				twice = append(twice, placed{r, p})
				continue
			}
			rules[r.ID] = placed{r, p}
		}
	}
	return rules, twice
}

// usedTwice says where an ID is found a second time.
func usedTwice(rules map[string]placed, again placed) string {
	first := rules[again.ID]
	return fmt.Sprintf("%s:%d: %s is also at %s:%d; an ID is used once", again.Path, again.Line+1, again.ID, first.Path, first.Line+1)
}

// Check compares the rules of head with base (nil when there is none) and lists what breaks G-CON-8.
// baseDay is the day of the base's last commit; a new rule is dated no earlier. A project's override may go away:
// the org rule then holds again.
func Check(base Files, baseDay string, head Files, today string) []string {
	now, twice := index(head)
	var problems []string
	for _, again := range twice {
		problems = append(problems, usedTwice(now, again))
	}
	for _, p := range sortedPaths(head) {
		doc := Parse(head[p])
		for _, n := range doc.NotRules {
			problems = append(problems, fmt.Sprintf("%s:%d: %s is in bold like a rule ID but is not read as a rule; write a rule at the start of a line as `- **%s** %s · <text>`, or drop the bold", p, n.Line+1, n.ID, n.ID, today))
		}
		for _, r := range doc.Rules {
			problems = append(problems, checkDates(p, r, today)...)
		}
	}
	if base == nil {
		return problems
	}
	before, _ := index(base)
	for _, id := range sortedIDs(now) {
		cur := now[id]
		if _, ok := before[id]; !ok && cur.Added != "" && cur.Added < baseDay {
			problems = append(problems, fmt.Sprintf("%s:%d: %s is new but dated %s, before the base's last commit on %s; a new rule carries the day it is added, %s", cur.Path, cur.Line+1, id, cur.Added, baseDay, today))
		}
	}
	for _, id := range sortedIDs(before) {
		old := before[id]
		cur, ok := now[id]
		if !ok && old.Override {
			continue
		}
		if !ok {
			problems = append(problems, fmt.Sprintf("%s:%d: %s is gone; a removed rule stays listed with the text %q", old.Path, old.Line+1, id, Removed))
			continue
		}
		where := fmt.Sprintf("%s:%d", cur.Path, cur.Line+1)
		if old.Added != "" && cur.Added != old.Added {
			problems = append(problems, fmt.Sprintf("%s: %s was added %s; that date never changes", where, id, old.Added))
		}
		switch {
		case old.Text == cur.Text:
			if old.Added != "" && changedDate(cur.Rule) != changedDate(old.Rule) {
				problems = append(problems, fmt.Sprintf("%s: %s kept its text, so its dates stay `%s`", where, id, datesOf(old.Rule)))
			}
		case old.Text == Removed && !old.Override:
			problems = append(problems, fmt.Sprintf("%s: %s was removed; an ID is never reused, give the rule a new ID", where, id))
		case !changeDated(old.Rule, cur.Rule, today):
			problems = append(problems, fmt.Sprintf("%s: %s changed its text; set its changed date to %s", where, id, today))
		}
	}
	return problems
}

// changeDated says whether cur, whose text differs from old, carries the day of that change:
// today, or a day later than old's last date. With OmitSameDayChanged, a rule added today needs no changed date.
func changeDated(old, cur Rule, today string) bool {
	last := cur.Changed
	if last == "" && OmitSameDayChanged && cur.Added == today {
		last = today
	}
	since := old.Changed
	if since == "" {
		since = old.Added
	}
	return last != "" && (last == today || last > since)
}

// datesOf is how r's dates read on its line.
func datesOf(r Rule) string {
	if c := changedDate(r); c != "" {
		return r.Added + " · changed " + c
	}
	return r.Added
}

// checkDates says what is wrong with a rule's own dates.
func checkDates(path string, r Rule, today string) []string {
	where := fmt.Sprintf("%s:%d", path, r.Line+1)
	if r.Added == "" {
		return []string{fmt.Sprintf("%s: %s has no date; write it as `- **%s** %s · <text>`", where, r.ID, r.ID, today)}
	}
	var problems []string
	for _, d := range []string{r.Added, r.Changed} {
		if d == "" {
			continue
		}
		t, err := time.Parse(time.DateOnly, d)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %s has %s, which is not a date", where, r.ID, d))
			continue
		}
		// A day of slack: the runner's day may be behind the writer's.
		if limit, _ := time.Parse(time.DateOnly, today); t.After(limit.AddDate(0, 0, 1)) {
			problems = append(problems, fmt.Sprintf("%s: %s has %s, which is after today", where, r.ID, d))
		}
	}
	if r.Changed != "" && r.Changed < r.Added {
		problems = append(problems, fmt.Sprintf("%s: %s was changed %s, before it was added %s", where, r.ID, r.Changed, r.Added))
	}
	if OmitSameDayChanged && r.Changed != "" && r.Changed == r.Added {
		problems = append(problems, fmt.Sprintf("%s: %s was changed the day it was added; leave changed out: `- **%s** %s · <text>`", where, r.ID, r.ID, r.Added))
	}
	return problems
}
