package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// geminiBody wraps text the way Gemini wraps model output, so tests build
// real response JSON without hand-escaping quotes.
func geminiBody(t *testing.T, text string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"candidates": []any{map[string]any{
			"content": map[string]any{
				"role":  "model",
				"parts": []any{map[string]any{"text": text}},
			},
		}},
	})
	if err != nil {
		t.Fatalf("marshaling fixture: %v", err)
	}
	return string(raw)
}

// askWithResponse runs Ask against a stub server returning status/body.
// The real API is never contacted.
func askWithResponse(t *testing.T, status int, body string) (AskResult, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-key", "gemini-2.5-flash")
	c.baseURL = srv.URL
	return c.Ask(context.Background(), "who knows machine learning", testRecords())
}

func TestAskParsesGeminiResponse(t *testing.T) {
	var gotPath, gotKey, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.URL.Query().Get("key")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		io.WriteString(w, geminiBody(t, `{"relevant": true, "answer": "Piyush builds ML backends."}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-key", "gemini-2.5-flash")
	c.baseURL = srv.URL

	got, err := c.Ask(context.Background(), "who knows machine learning", testRecords())
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !got.Relevant {
		t.Error("Relevant = false, want true")
	}
	if got.Answer != "Piyush builds ML backends." {
		t.Errorf("Answer = %q", got.Answer)
	}

	if want := "/v1beta/models/gemini-2.5-flash:generateContent"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if gotKey != "test-key" {
		t.Errorf("api key not passed as ?key= query param, got %q", gotKey)
	}
	if !strings.Contains(gotBody, `"contents"`) || !strings.Contains(gotBody, "who knows machine learning") {
		t.Errorf("request body missing prompt contents: %s", gotBody)
	}
}

func TestAskStripsMarkdownFences(t *testing.T) {
	got, err := askWithResponse(t, http.StatusOK,
		geminiBody(t, "```json\n{\"relevant\": false}\n```"))
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if got.Relevant {
		t.Error("Relevant = true, want false")
	}
}

func TestAskRejectsNonJSONOutput(t *testing.T) {
	_, err := askWithResponse(t, http.StatusOK,
		geminiBody(t, "I could not find anything relevant."))
	if err == nil {
		t.Fatal("want error for non-JSON output, got nil")
	}
	if !strings.Contains(err.Error(), "not valid JSON") {
		t.Errorf("error = %q, want mention of invalid JSON", err)
	}
}

func TestAskRejectsMissingRelevantField(t *testing.T) {
	_, err := askWithResponse(t, http.StatusOK,
		geminiBody(t, `{"answer": "Piyush builds ML backends."}`))
	if err == nil {
		t.Fatal("want error for missing 'relevant' field, got nil")
	}
	if !strings.Contains(err.Error(), "relevant") {
		t.Errorf("error = %q, want mention of the relevant field", err)
	}
}

func TestAskReportsGeminiErrorBody(t *testing.T) {
	_, err := askWithResponse(t, http.StatusBadRequest,
		`{"error":{"code":400,"message":"API key not valid.","status":"INVALID_ARGUMENT"}}`)
	if err == nil {
		t.Fatal("want error for HTTP 400, got nil")
	}
	if !strings.Contains(err.Error(), "API key not valid") || !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %q, want Google's message and the status", err)
	}
}

func TestAskRequiresAPIKey(t *testing.T) {
	c := NewClient("", "gemini-2.5-flash")
	_, err := c.Ask(context.Background(), "anything", nil)
	if err == nil {
		t.Fatal("want error for empty API key, got nil")
	}
	if !strings.Contains(err.Error(), "LLM_API_KEY") {
		t.Errorf("error = %q, want mention of LLM_API_KEY", err)
	}
}

func TestStripFences(t *testing.T) {
	cases := map[string]string{
		"plain":                  `{"relevant": true}`,
		"json fence":             "```json\n{\"relevant\": true}\n```",
		"bare fence":             "```\n{\"relevant\": true}\n```",
		"compact fence":          "```json{\"relevant\": true}```",
		"surrounding whitespace": "  \n{\"relevant\": false}\n  ",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			got := stripFences(in)
			if !strings.HasPrefix(got, "{") || !strings.HasSuffix(got, "}") {
				t.Errorf("stripFences(%q) = %q, want bare JSON", in, got)
			}
		})
	}
}
