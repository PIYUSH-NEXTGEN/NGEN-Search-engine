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
	prompt := BuildPrompt("who knows machine learning", testRecords())

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
	prompt := BuildPrompt("anything", nil)
	if !strings.Contains(prompt, "[]") {
		t.Errorf("empty records should render as [], got prompt:\n%s", prompt)
	}
}
