package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/steipete/sag/internal/elevenlabs"
)

func TestVoicesCommand(t *testing.T) {
	resetRootCommandState()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/voices" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"voices":[{"voice_id":"id1","name":"Alpha","category":"premade"}]}`))
	}))
	defer srv.Close()

	cfg.APIKey = "key"
	cfg.BaseURL = srv.URL

	restore, readOut := captureStdoutVoices(t)
	defer restore()

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"voices", "--limit", "1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rootCmd execute: %v", err)
	}

	out := buf.String() + readOut()
	if !bytes.Contains([]byte(out), []byte("VOICE ID")) {
		t.Fatalf("expected table output, got %q", out)
	}

	// reset args to avoid polluting other tests
	rootCmd.SetArgs(nil)
	_ = os.Unsetenv("ELEVENLABS_API_KEY")
}

func TestFilterVoicesByName(t *testing.T) {
	voices := []elevenlabs.Voice{
		{VoiceID: "id1", Name: "Sarah"},
		{VoiceID: "id2", Name: "Roger - Casual"},
		{VoiceID: "id3", Name: "ROGUE"},
	}

	filtered := filterVoicesByName(voices, "rog")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 voices, got %d", len(filtered))
	}
	if filtered[0].VoiceID != "id2" || filtered[1].VoiceID != "id3" {
		t.Fatalf("unexpected filter order: %+v", filtered)
	}
}

func TestVoicesCommandTryRequiresFilter(t *testing.T) {
	resetRootCommandState()

	cfg.APIKey = "key"
	cfg.BaseURL = "http://example.invalid"
	t.Cleanup(func() {
		cfg.APIKey = ""
		cfg.BaseURL = ""
	})

	voicesCmd, _, err := rootCmd.Find([]string{"voices"})
	if err != nil {
		t.Fatalf("find voices command: %v", err)
	}
	for _, name := range []string{"limit", "search", "query", "label", "try"} {
		flag := voicesCmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("missing %s flag", name)
		}
		switch name {
		case "limit":
			if err := flag.Value.Set("100"); err != nil {
				t.Fatalf("reset limit: %v", err)
			}
		case "search":
			if err := flag.Value.Set(""); err != nil {
				t.Fatalf("reset search: %v", err)
			}
		case "query":
			if err := flag.Value.Set(""); err != nil {
				t.Fatalf("reset query: %v", err)
			}
		case "label":
			if err := flag.Value.Set(""); err != nil {
				t.Fatalf("reset label: %v", err)
			}
		case "try":
			if err := flag.Value.Set("false"); err != nil {
				t.Fatalf("reset try: %v", err)
			}
		}
		flag.Changed = false
	}

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"voices", "--try"})

	err = rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--try requires") {
		t.Fatalf("expected --try requires error, got %v", err)
	}

	rootCmd.SetArgs(nil)
}

func TestParseLabelFilters(t *testing.T) {
	filters, err := parseLabelFilters([]string{"Accent=British", "use_case=Character"})
	if err != nil {
		t.Fatalf("parseLabelFilters error: %v", err)
	}
	if len(filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(filters))
	}
	if filters[0].key != "accent" || filters[0].value != "british" {
		t.Fatalf("unexpected filter: %+v", filters[0])
	}
}

func TestFilterVoicesByLabels(t *testing.T) {
	voices := []elevenlabs.Voice{
		{VoiceID: "id1", Name: "Alpha", Labels: map[string]string{"accent": "British", "gender": "male"}},
		{VoiceID: "id2", Name: "Beta", Labels: map[string]string{"accent": "American"}},
		{VoiceID: "id3", Name: "Gamma", Labels: map[string]string{"Accent": "British"}},
	}
	filters, err := parseLabelFilters([]string{"accent=british"})
	if err != nil {
		t.Fatalf("parseLabelFilters error: %v", err)
	}
	filtered := filterVoicesByLabels(voices, filters)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 voices, got %d", len(filtered))
	}
	if filtered[0].VoiceID != "id1" || filtered[1].VoiceID != "id3" {
		t.Fatalf("unexpected filtered order: %+v", filtered)
	}
}

func TestRankVoicesByQuery(t *testing.T) {
	voices := []elevenlabs.Voice{
		{VoiceID: "id1", Name: "Calm Narrator", Description: "Relaxed, smooth storyteller"},
		{VoiceID: "id2", Name: "Mad Lab", Description: "Crazy scientist with wild energy", Labels: map[string]string{"use_case": "character"}},
		{VoiceID: "id3", Name: "Plain Voice", Description: "Neutral"},
	}
	ranked := rankVoicesByQuery(voices, "crazy scientist")
	if len(ranked) == 0 {
		t.Fatalf("expected ranked voices, got none")
	}
	if ranked[0].VoiceID != "id2" {
		t.Fatalf("expected id2 first, got %s", ranked[0].VoiceID)
	}
}

func captureStdoutVoices(t *testing.T) (restore func(), read func() string) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	return func() {
			_ = w.Close()
			os.Stdout = orig
		}, func() string {
			_ = w.Close()
			b, _ := io.ReadAll(r)
			return string(b)
		}
}
