package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
)

// BuildPrompt renders the single user turn for one Ask: the response rules,
// any prior turns from the caller's session (so follow-ups such as "tell me
// more about her" resolve), the query, and the retrieved members as JSON so
// the model reasons over structured data instead of prose. history may be
// nil. It returns a plain string so the grounding rules can be asserted in
// tests without any network call.
func BuildPrompt(query string, records []search.Result, history []Turn) string {
	if records == nil {
		// Marshal an empty slice so the prompt shows [] rather than null.
		records = []search.Result{}
	}
	data, err := json.Marshal(records)
	if err != nil {
		// search.Result is all plain fields, so this can't realistically fire —
		// but a prompt must never be built from half-written JSON.
		data = []byte("[]")
	}

	var b strings.Builder
	b.WriteString(`You are the answer layer of a community search engine.
Given a user query and the member records retrieved for it, reply with ONLY
raw JSON — no markdown fences, no preamble, no text before or after it. The
JSON must be exactly one of these two shapes:
{"relevant": true, "answer": "a flowing paragraph"}
{"relevant": false}

Rules:
- If the query is not about this community — not about members, skills,
  roles, projects, or experience — reply {"relevant": false}.
- If the records below do not support an answer, reply {"relevant": false}.
- Otherwise set "relevant": true and write "answer" as one flowing
  paragraph, not a list.
- Use only facts present in the records. Do not invent names, skills,
  employers, locations, or any other detail that is not in the data.
`)
	if len(history) > 0 {
		b.WriteString(`
Previous turns — context for follow-up questions such as "tell me more about
her". They are not new facts: the grounding rules above still apply.
`)
		for _, turn := range history {
			fmt.Fprintf(&b, "User: %s\nAssistant: %s\n", turn.Query, turn.Answer)
		}
	}
	fmt.Fprintf(&b, `
Query:
%s

Retrieved records (JSON):
%s`, query, data)
	return b.String()
}
