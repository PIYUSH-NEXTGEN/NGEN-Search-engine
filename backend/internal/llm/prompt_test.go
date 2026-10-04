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

func TestBuildPromptCarriesQueryRecordsAndRules(t *testing.T) {
	prompt := BuildPrompt("who knows machine learning", testRecords(), nil)

	wants := []string{
		"who knows machine learning", // the query itself
		`"full_name"`,                // records as structured JSON, not prose
		"Piyush Baraskar",            // every record's data
		"Divynash Shakya",
		"raw JSON",                 // JSON-only instruction
		`{"relevant": true`,        // exact response shape
		"flowing paragraph",        // paragraph, not a list
		"Do not invent",            // no facts beyond the records
		"not about this community", // rejection rule
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Errorf("BuildPrompt output missing %q", want)
		}
	}
}

func TestBuildPromptEmptyRecordsRenderAsArray(t *testing.T) {
	prompt := BuildPrompt("anything", nil, nil)
	if !strings.Contains(prompt, "[]") {
		t.Errorf("empty records should render as [], got prompt:\n%s", prompt)
	}
}

func TestBuildPromptIncludesPriorTurns(t *testing.T) {
	history := []Turn{
		{Query: "who works in cybersecurity", Answer: "Divynash Shakya studies cybersecurity."},
	}
	prompt := BuildPrompt("tell me more about her", testRecords(), history)

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
	prompt := BuildPrompt("who knows machine learning", testRecords(), nil)
	if strings.Contains(prompt, "Previous turns") {
		t.Error("no history passed, but prompt still contains a Previous turns section")
	}
}

func TestBuildPromptAllowsAnswerFromHistoryWhenRecordsEmpty(t *testing.T) {
	history := []Turn{
		{Query: "machine learning", Answer: "Piyush Baraskar is based in Bhopal."},
	}
	prompt := BuildPrompt("where are they based?", nil, history)

	wants := []string{
		"Previous turns",                                   // history section exists
		"Piyush Baraskar is based in Bhopal.",              // prior answer's facts usable
		"neither the records below nor the previous turns", // refusal only when both are empty of support
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}
