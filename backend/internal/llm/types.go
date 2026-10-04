// Package llm answers a community query from the members retrieval already
// found for it. Callers pass a query plus records and get a grounded
// yes/no back — nothing here touches the database, Redis, or HTTP handlers,
// so the whole flow can be exercised without a running stack.
package llm

// AskResult is one Ask outcome: whether the query belongs to this community
// at all, and the generated paragraph when it does.
type AskResult struct {
	Relevant bool   `json:"relevant"`
	Answer   string `json:"answer"`
}

// The wire shapes below mirror Google's generateContent REST API — only the
// fields we actually read or write.
// https://ai.google.dev/gemini-api/docs/text-generation

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

// geminiError is the body Google sends alongside non-200 responses.
type geminiError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
