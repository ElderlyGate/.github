package main

import (
	"strings"
	"testing"
)

const today = "2026-11-03"

func file(lines ...string) Files {
	return Files{"constitution/tech-stack.md": strings.Join(lines, "\n") + "\n"}
}

func expectProblems(t *testing.T, problems []string, want ...string) {
	t.Helper()
	if len(problems) != len(want) {
		t.Fatalf("problems = %q, want %d", problems, len(want))
	}
	for i, w := range want {
		if !strings.Contains(problems[i], w) {
			t.Errorf("problem %d = %q, want it to say %q", i, problems[i], w)
		}
	}
}

func TestCheckPassesDatedRulesAndAChangeWithItsDate(t *testing.T) {
	base := file("- **G-CON-1** 2026-10-27 · Old words.", "- **G-CON-2** 2026-10-27 · Same.")
	head := file("- **G-CON-1** 2026-10-27 · changed 2026-11-03 · New words.", "- **G-CON-2** 2026-10-27 · Same.", "- **G-CON-3** 2026-11-03 · A new rule.")
	expectProblems(t, Check(base, "", head, today))
}

func TestCheckFailsATextChangeWithoutItsChangedDate(t *testing.T) {
	base := file("- **G-CON-1** 2026-10-27 · changed 2026-10-28 · Old words.")
	head := file("- **G-CON-1** 2026-10-27 · changed 2026-10-28 · New words.")
	expectProblems(t, Check(base, "", head, today), "constitution/tech-stack.md:1: G-CON-1 changed its text; set its changed date to 2026-11-03")
}

func TestCheckFailsANewRuleWithoutADate(t *testing.T) {
	expectProblems(t, Check(file(), "", file("- **G-CON-8** Dates."), today), "G-CON-8 has no date")
}

func TestCheckWithoutABaseChecksDatesOnly(t *testing.T) {
	head := file("- **G-CON-1** 2026-10-28 · changed 2026-10-27 · Words.", "- **G-CON-2** 2026-12-01 · Words.", "- **G-CON-3** 2026-13-01 · Words.")
	expectProblems(t, Check(nil, "", head, today), "before it was added", "after today", "not a date")
}

func TestCheckFailsAMovedAddedDate(t *testing.T) {
	expectProblems(t, Check(file("- **V-1** 2026-10-27 · Words."), "", file("- **V-1** 2026-10-28 · Words."), today), "was added 2026-10-27")
}

func TestCheckFailsARemovedRuleThatIsGoneOrReused(t *testing.T) {
	base := file("- **V-1** 2026-10-27 · Words.", "- **V-2** 2026-10-27 · changed 2026-10-30 · Removed.")
	head := file("- **V-2** 2026-10-27 · changed 2026-11-03 · Something else.")
	expectProblems(t, Check(base, "", head, today), "V-1 is gone", "V-2 was removed; an ID is never reused")
}

func TestCheckFailsAnIDUsedTwice(t *testing.T) {
	head := Files{
		"constitution/mission.md":    "- **P-1** 2026-10-27 · One.\n",
		"constitution/tech-stack.md": "- **P-1** 2026-10-27 · Two.\n",
	}
	expectProblems(t, Check(nil, "", head, today), "constitution/tech-stack.md:1: P-1 is also at constitution/mission.md:1")
}

func TestCheckAcceptsTheBackfillOfAnUndatedBase(t *testing.T) {
	base := file("- **V-1** Words.", "- **V-2** More words.")
	head := file("- **V-1** 2026-10-27 · Words.", "- **V-2** 2026-10-27 · changed 2026-10-28 · More words.")
	expectProblems(t, Check(base, "", head, today))
}

func TestCheckFollowsARuleIntoAnotherFile(t *testing.T) {
	base := Files{"constitution/mission.md": "- **P-1** 2026-10-27 · Words.\n"}
	head := Files{"constitution/mission.md": "", "constitution/tech-stack.md": "- **P-1** 2026-10-27 · Words.\n"}
	expectProblems(t, Check(base, "", head, today))
}

func TestCheckLetsAProjectDropAnOverride(t *testing.T) {
	base := file("## Overrides", "", "- **G-STACK-6** 2026-10-27 · (type the name). Delete asks for the name.", "- **G-STACK-8** 2026-10-27 · Paths.")
	head := file("## Overrides", "", "- **G-STACK-8** 2026-10-27 · Paths.")
	expectProblems(t, Check(base, "", head, today))
}

func TestCheckFailsAChangedDateThatIsNotLaterThanTheBase(t *testing.T) {
	base := file("- **V-1** 2026-10-27 · changed 2026-10-30 · Old words.")
	backdated := file("- **V-1** 2026-10-27 · changed 2026-10-29 · New words.")
	expectProblems(t, Check(base, "", backdated, today), "V-1 changed its text; set its changed date to 2026-11-03")
	kept := file("- **V-1** 2026-10-27 · changed 2026-10-30 · New words.")
	expectProblems(t, Check(base, "", kept, today), "V-1 changed its text")
	sameDayAgain := file("- **V-1** 2026-10-27 · changed 2026-11-03 · Newer words.")
	expectProblems(t, Check(file("- **V-1** 2026-10-27 · changed 2026-11-03 · New words."), "", sameDayAgain, today))
}

func TestCheckFailsMovedChangedDatesOnTheSameText(t *testing.T) {
	base := file("- **V-1** 2026-10-27 · changed 2026-10-28 · Words.", "- **V-2** 2026-10-27 · Words.")
	head := file("- **V-1** 2026-10-27 · Words.", "- **V-2** 2026-10-27 · changed 2026-11-03 · Words.")
	expectProblems(t, Check(base, "", head, today),
		"V-1 kept its text, so its dates stay `2026-10-27 · changed 2026-10-28`",
		"V-2 kept its text, so its dates stay `2026-10-27`")
}

func TestCheckFailsANewRuleDatedBeforeTheBase(t *testing.T) {
	base := file("- **V-1** 2026-10-27 · Words.")
	head := file("- **V-1** 2026-10-27 · Words.", "- **V-2** 2026-10-30 · New.", "- **V-3** 2026-11-01 · New.")
	expectProblems(t, Check(base, "2026-11-01", head, today),
		"constitution/tech-stack.md:2: V-2 is new but dated 2026-10-30, before the base's last commit on 2026-11-01")
}
