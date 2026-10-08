package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
)

// Info is one community_info row as the answer layer sees it — the llm
// package's own view of the standing context, so prompt and client don't
// depend on the search package's row shape.
type Info struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// BuildPrompt renders the single user turn for one Ask: the response rules,
// any prior turns from the caller's session (so follow-ups such as "tell me
// more about her" resolve), the query, the standing community info, and the
// grouped retrieval as labelled JSON sections so the model reasons over
// structured data instead of prose. history may be nil. It returns a plain
// string so the grounding rules can be asserted in tests without any network
// call.
//
// info is the full community_info table — always present, so general
// questions like "what is this community about?" work even when full-text
// search matched nothing. It is the only way community info enters the
// prompt: results carries just the per-query member matches (SearchAll
// never searches community_info), so the standing rows appear exactly once.
func BuildPrompt(query string, results search.AllResults, info []Info, history []Turn) string {
	members := results.Members
	if members == nil {
		// Marshal an empty slice so the prompt shows [] rather than null.
		members = []search.Result{}
	}
	if info == nil {
		info = []Info{}
	}

	var b strings.Builder
	b.WriteString(`You are the answer layer of a community search engine.
Given a user query, the community's standing info, and the member records
retrieved for the query, reply with ONLY raw JSON — no markdown fences, no
preamble, no text before or after it. The JSON must be exactly one of these
two shapes:
{"relevant": true, "answer": "a flowing paragraph"}
{"relevant": false}

Rules:
- A query is relevant if it's about this community, its members, its rules,
  how to join, or what the community offers. Anything else returns
  {"relevant": false}. Off-topic queries (weather, general trivia, coding
  help unrelated to this community) must still be rejected with
  {"relevant": false} even though community info is always present in the
  prompt — the standing info below never makes an off-topic query relevant.
- Answer using ONLY facts present in the provided sections. Never invent
  members, rules, links, dates, events, channels, numbers or any other
  detail that appears nowhere below. If the question is relevant but the
  provided data doesn't contain the answer, set "relevant": true and say
  clearly that the community data doesn't cover it, rather than guessing.
- When the answer is a rule, state it accurately and faithfully as it
  appears in the data: do not soften, loosen or reword it, and keep any
  penalty exactly (for example, "immediate ban").
- When asked how to join, include the invite link exactly as it appears in
  the community info — never a modified or guessed link.
- Otherwise set "relevant": true and write "answer" as one flowing
  paragraph, not a list. A numbered list is acceptable only when the user
  asks for all the rules.
`)
	if len(history) > 0 {
		b.WriteString(`
Previous turns — context for follow-up questions such as "tell me more about
her". Facts stated in an earlier answer were grounded in that turn's records
and may be used to answer a follow-up about them; still never invent names
or details that appear nowhere above.
`)
		for _, turn := range history {
			fmt.Fprintf(&b, "User: %s\nAssistant: %s\n", turn.Query, turn.Answer)
		}
	}
	fmt.Fprintf(&b, `
Community info (always provided):
%s

Matching members (JSON):
%s

Query:
%s`, mustMarshal(info), mustMarshal(members), query)
	return b.String()
}

// mustMarshal renders one prompt section as JSON. These shapes are all plain
// fields, so this can't realistically fail — but a prompt must never be
// built from half-written JSON, so fall back to [] on error.
func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte("[]")
	}
	return data
}
