package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
)

// geminiBaseURL is Google's generativelanguage endpoint. Tests point
// Client.baseURL at an httptest server instead — no test ever calls the
// real API.
const geminiBaseURL = "https://generativelanguage.googleapis.com"

// maxResponseBytes caps how much of the response we will read.
const maxResponseBytes = 1 << 20 // 1 MiB

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string // overridable so tests never reach the real API
}

// NewClient builds a client for the Gemini generateContent API.
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			// api.NewRouter times the whole request out at 10s, so a longer
			// client timeout here would never be the one to fire first.
			Timeout: 10 * time.Second,
		},
		baseURL: geminiBaseURL,
	}
}

// Ask sends the query and records to Gemini and parses the JSON answer.
// Anything the model gets wrong comes back as an error, never a panic.
func (c *Client) Ask(ctx context.Context, query string, records []search.Result) (AskResult, error) {
	if strings.TrimSpace(query) == "" {
		return AskResult{}, fmt.Errorf("query must not be empty")
	}
	if c.apiKey == "" {
		return AskResult{}, fmt.Errorf("llm: LLM_API_KEY is not set")
	}

	payload, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{
			Role:  "user",
			Parts: []geminiPart{{Text: BuildPrompt(query, records)}},
		}},
	})
	if err != nil {
		return AskResult{}, fmt.Errorf("marshaling gemini request: %w", err)
	}

	// This endpoint takes the API key as a query parameter, not a header.
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s",
		c.baseURL, url.PathEscape(c.model), url.QueryEscape(c.apiKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return AskResult{}, fmt.Errorf("building gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AskResult{}, fmt.Errorf("calling gemini: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return AskResult{}, fmt.Errorf("reading gemini response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return AskResult{}, fmt.Errorf("gemini: %s", geminiStatusError(body, resp.StatusCode))
	}

	var parsed geminiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return AskResult{}, fmt.Errorf("parsing gemini response: %w", err)
	}

	text, err := textFromResponse(parsed)
	if err != nil {
		return AskResult{}, err
	}
	return parseAskResult(text)
}

// textFromResponse pulls candidates[0].content.parts[*].text.
func textFromResponse(r geminiResponse) (string, error) {
	if len(r.Candidates) == 0 {
		return "", fmt.Errorf("gemini returned no candidates (the prompt may have been blocked)")
	}
	var b strings.Builder
	for _, part := range r.Candidates[0].Content.Parts {
		b.WriteString(part.Text)
	}
	if b.String() == "" {
		return "", fmt.Errorf("gemini returned an empty completion")
	}
	return b.String(), nil
}

// geminiStatusError prefers Google's own message when the body carries one,
// so an invalid key reads as something actionable instead of a bare 400.
func geminiStatusError(body []byte, status int) string {
	var parsed geminiError
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Message != "" {
		return fmt.Sprintf("%s (HTTP %d)", parsed.Error.Message, status)
	}
	return fmt.Sprintf("HTTP %d", status)
}

// parseAskResult strips any accidental ```json fence and requires the exact
// shape the prompt asked for, so a surprise from the model surfaces as an
// error rather than a silently-empty answer.
func parseAskResult(text string) (AskResult, error) {
	var wire struct {
		Relevant *bool  `json:"relevant"`
		Answer   string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(stripFences(text)), &wire); err != nil {
		return AskResult{}, fmt.Errorf("model output is not valid JSON: %w", err)
	}
	if wire.Relevant == nil {
		return AskResult{}, fmt.Errorf(`model output is missing the required field "relevant"`)
	}
	return AskResult{Relevant: *wire.Relevant, Answer: wire.Answer}, nil
}

// stripFences removes a wrapping ```json fence when the model adds one
// despite the prompt, in both the usual multi-line and compact forms.
func stripFences(s string) string {
	out := strings.TrimSpace(s)
	if !strings.HasPrefix(out, "```") {
		return out
	}
	rest := strings.TrimPrefix(out, "```")
	if i := strings.IndexByte(rest, '\n'); i >= 0 {
		// The language tag sits on its own line; drop it too.
		rest = rest[i+1:]
	} else {
		rest = strings.TrimPrefix(rest, "json")
		rest = strings.TrimPrefix(rest, "JSON")
	}
	rest = strings.TrimSpace(rest)
	return strings.TrimSpace(strings.TrimSuffix(rest, "```"))
}
