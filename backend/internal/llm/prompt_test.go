package llm

import (
	"strings"
	"testing"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
)

func testRecords() []search.Result {
	return []search.Result{
		{ID: "66d7", FullName: "Piyush Baraskar", Headline: "Machine learning & Backend developer", Bio: "Builds ML backends", Location: "Bhopal", Rank: 0.9},
		{ID: "6e77", FullName: "Divynash Shakya", Headline: "Cybersecurity engineer", Bio: "B.Tech CSE student", Location: "Bhopal", Rank: 0.1},
	}
}

func testAllResults() search.AllResults {
	return search.AllResults{Members: testRecords()}
}

func testStanding() []Info {
	return []Info{
		{Slug: "about", Title: "About", Body: "A community of builders."},
		{Slug: "mission", Title: "Mission", Body: "Connect builders."},
		{Slug: "rules", Title: "Community Rules", Body: "Be kind. Spamming leads to an immediate ban."},
		{Slug: "how-to-join", Title: "How to Join", Body: "Join on Discord: https://discord.gg/AUz7KqDrnf"},
		{Slug: "what-you-can-do", Title: "What you can do", Body: "Find people."},
	}
}

func TestBuildPromptCarriesQueryRecordsAndRules(t *testing.T) {
	prompt := BuildPrompt("who knows machine learning", testAllResults(), testStanding(), nil)

	wants := []string{
		"who knows machine learning", // the query itself
		`"full_name"`,                // members as structured JSON, not prose
		"Piyush Baraskar",            // every group renders its data
		"Divynash Shakya",
		"Community info (always provided)", // standing section label
		"Matching members",                 // members section label
		"A community of builders.",         // standing community info always present
		"Find people.",
		"raw JSON",          // JSON-only instruction
		`{"relevant": true`, // exact response shape
		"flowing paragraph", // paragraph, not a list
		"Never invent",      // no facts beyond the sections
		"doesn't cover it",  // relevant-but-uncovered rule
		"Off-topic queries", // off-topic rejection despite standing info
		"weather",
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Errorf("BuildPrompt output missing %q", want)
		}
	}
	if strings.Contains(prompt, "Matching projects") {
		t.Error("prompt contains a Matching projects section, but projects were removed")
	}
}

func TestBuildPromptEmptyRecordsRenderAsArray(t *testing.T) {
	prompt := BuildPrompt("anything", search.AllResults{}, nil, nil)
	if got := strings.Count(prompt, "[]"); got < 2 {
		t.Errorf("empty groups should each render as [], got %d in prompt:\n%s", got, prompt)
	}
}

func TestBuildPromptStandingInfoPresentWithoutMatches(t *testing.T) {
	// General questions must work even when full-text search finds nothing:
	// standing info is present while all matched groups are empty.
	prompt := BuildPrompt("what is this community about", search.AllResults{}, testStanding(), nil)

	for _, want := range []string{"Community info", "A community of builders.", "Matching members", "[]"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPromptIncludesPriorTurns(t *testing.T) {
	history := []Turn{
		{Query: "who works in cybersecurity", Answer: "Divynash Shakya studies cybersecurity."},
	}
	prompt := BuildPrompt("tell me more about her", testAllResults(), testStanding(), history)

	wants := []string{
		"Previous turns",                         // history section exists
		"who works in cybersecurity",             // the prior question
		"Divynash Shakya studies cybersecurity.", // the prior answer
		"tell me more about her",                 // current query still present
		"Facts stated in an earlier answer",      // prior answers are usable grounding
		"never invent names",                     // history can't loosen grounding
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPromptOmitsHistorySectionWhenNone(t *testing.T) {
	prompt := BuildPrompt("who knows machine learning", testAllResults(), testStanding(), nil)
	if strings.Contains(prompt, "Previous turns") {
		t.Error("no history passed, but prompt still contains a Previous turns section")
	}
}

func TestBuildPromptAllowsAnswerFromHistoryWhenRecordsEmpty(t *testing.T) {
	history := []Turn{
		{Query: "machine learning", Answer: "Piyush Baraskar is based in Bhopal."},
	}
	prompt := BuildPrompt("where are they based?", search.AllResults{}, testStanding(), history)

	wants := []string{
		"Previous turns",                      // history section exists
		"Piyush Baraskar is based in Bhopal.", // prior answer's facts usable
		"A community of builders.",            // standing info still present
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPromptIncludesEveryCommunityInfoRow(t *testing.T) {
	// Every row of the tiny community_info table must land in the prompt —
	// no per-query filtering of the standing context.
	standing := testStanding()
	prompt := BuildPrompt("how do I join", search.AllResults{}, standing, nil)

	for _, row := range standing {
		if !strings.Contains(prompt, row.Body) {
			t.Errorf("prompt missing body of row %q: %s", row.Slug, prompt)
		}
		if !strings.Contains(prompt, row.Slug) {
			t.Errorf("prompt missing slug %q", row.Slug)
		}
	}
}

func TestBuildPromptPassesInviteLinkThroughUnchanged(t *testing.T) {
	// The invite link must reach the model byte-for-byte: the link-handling
	// rule plus the raw data section, so the answer can quote it exactly.
	prompt := BuildPrompt("how do I join", search.AllResults{}, testStanding(), nil)

	wants := []string{
		"https://discord.gg/AUz7KqDrnf",      // the link, verbatim
		"never a modified or guessed link",   // link-handling rule
		"how to join",                        // relevant-topic rule mentions joining
		"do not soften, loosen or reword it", // rule-fidelity rule
		"immediate ban",                      // penalty wording stays in the data
		"asks for all the rules",             // numbered-list exception
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPromptRelevanceCoversRulesAndJoining(t *testing.T) {
	// The relevance rule itself must name rules and how-to-join as on-topic,
	// otherwise the model could reject "what are the rules?" as off-topic.
	prompt := BuildPrompt("what are the rules", search.AllResults{}, testStanding(), nil)
	for _, want := range []string{"its rules,", "how to join,"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("relevance rule missing %q in prompt:\n%s", want, prompt)
		}
	}
}
