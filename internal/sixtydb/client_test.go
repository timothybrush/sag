package sixtydb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/steipete/sag/internal/tts"
)

func TestNewClientDefaultsBase(t *testing.T) {
	c := NewClient("key", "")
	if c.baseURL != DefaultBaseURL {
		t.Fatalf("unexpected baseURL: %s", c.baseURL)
	}
}

func TestListVoicesUnwrapsData(t *testing.T) {
	desc := "warm narrator"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/myvoices" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":[
			{"voice_id":"v1","name":"Aria","category":"professional","model":"60db Quality","labels":{"gender":"female","accent":"American"},"description":"` + desc + `"},
			{"voice_id":"v2","name":"Ravi","category":"cloned","model":"60db Fast","labels":{"gender":"male"},"description":null}
		]}`))
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	voices, err := c.ListVoices(context.Background())
	if err != nil {
		t.Fatalf("ListVoices error: %v", err)
	}
	if len(voices) != 2 {
		t.Fatalf("expected 2 voices, got %d", len(voices))
	}
	if voices[0].VoiceID != "v1" || voices[0].Name != "Aria" || voices[0].Category != "professional" {
		t.Fatalf("unexpected voice[0]: %+v", voices[0])
	}
	if voices[0].Description != desc {
		t.Fatalf("expected description %q, got %q", desc, voices[0].Description)
	}
	if voices[0].Labels["model"] != "60db Quality" {
		t.Fatalf("expected model folded into labels, got %+v", voices[0].Labels)
	}
	if voices[1].Description != "" {
		t.Fatalf("expected empty description for null, got %q", voices[1].Description)
	}
}

func TestSearchVoicesFiltersAndLimits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":[
			{"voice_id":"v1","name":"Roger"},
			{"voice_id":"v2","name":"Rogue"},
			{"voice_id":"v3","name":"Sarah"}
		]}`))
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	voices, err := c.SearchVoices(context.Background(), "rog", 1)
	if err != nil {
		t.Fatalf("SearchVoices error: %v", err)
	}
	if len(voices) != 1 {
		t.Fatalf("expected 1 voice after limit, got %d", len(voices))
	}
	if voices[0].VoiceID != "v1" {
		t.Fatalf("expected v1, got %s", voices[0].VoiceID)
	}
}

func TestConvertTTSDecodesBase64AndTranslatesParams(t *testing.T) {
	want := []byte("decoded-audio-bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tts-synthesize" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["text"] != "hi" {
			t.Fatalf("expected text hi, got %v", body["text"])
		}
		if body["voice_id"] != "v1" {
			t.Fatalf("expected voice_id v1, got %v", body["voice_id"])
		}
		// 0.5 stability -> 50, 0.8 similarity -> 80 (0..1 -> 0..100)
		if body["stability"] != float64(50) {
			t.Fatalf("expected stability 50, got %v", body["stability"])
		}
		if body["similarity"] != float64(80) {
			t.Fatalf("expected similarity 80, got %v", body["similarity"])
		}
		if body["speed"] != 1.1 {
			t.Fatalf("expected speed 1.1, got %v", body["speed"])
		}
		// mp3_44100_128 -> mp3
		if body["output_format"] != "mp3" {
			t.Fatalf("expected output_format mp3, got %v", body["output_format"])
		}
		// ElevenLabs-only fields must not appear.
		for _, k := range []string{"model_id", "style", "use_speaker_boost", "seed", "language_code"} {
			if _, ok := body[k]; ok {
				t.Fatalf("expected %q to be absent from 60db body", k)
			}
		}
		resp := map[string]any{"success": true, "audio_base64": base64.StdEncoding.EncodeToString(want)}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	stability := 0.5
	similarity := 0.8
	speed := 1.1
	c := NewClient("key", srv.URL)
	data, err := c.ConvertTTS(context.Background(), "v1", tts.TTSRequest{
		Text:         "hi",
		ModelID:      "eleven_v3",
		OutputFormat: "mp3_44100_128",
		VoiceSettings: &tts.VoiceSettings{
			Stability:       &stability,
			SimilarityBoost: &similarity,
			Speed:           &speed,
		},
	})
	if err != nil {
		t.Fatalf("ConvertTTS error: %v", err)
	}
	if string(data) != string(want) {
		t.Fatalf("unexpected decoded audio: %q", string(data))
	}
}

func TestStreamTTSDecodesNDJSON(t *testing.T) {
	chunk1 := base64.StdEncoding.EncodeToString([]byte("hello-"))
	chunk2 := base64.StdEncoding.EncodeToString([]byte("world"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tts-stream" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		// streaming body omits output_format
		if _, ok := body["output_format"]; ok {
			t.Fatalf("expected output_format omitted from stream body")
		}
		_, _ = io.WriteString(w, `{"type":"chunk","result":{"audioContent":"`+chunk1+`"}}`+"\n")
		_, _ = io.WriteString(w, `{"type":"chunk","result":{"audioContent":"`+chunk2+`"}}`+"\n")
		_, _ = io.WriteString(w, `{"type":"complete"}`+"\n")
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	rc, err := c.StreamTTS(context.Background(), "v1", tts.TTSRequest{Text: "hi"}, 0)
	if err != nil {
		t.Fatalf("StreamTTS error: %v", err)
	}
	defer func() { _ = rc.Close() }()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if string(got) != "hello-world" {
		t.Fatalf("unexpected decoded stream: %q", string(got))
	}
}

func TestStreamTTSSurfacesErrorFrame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"type":"error","message":"voice not found"}`+"\n")
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	rc, err := c.StreamTTS(context.Background(), "v1", tts.TTSRequest{Text: "hi"}, 0)
	if err != nil {
		t.Fatalf("StreamTTS error: %v", err)
	}
	defer func() { _ = rc.Close() }()
	_, err = io.ReadAll(rc)
	if err == nil || !strings.Contains(err.Error(), "voice not found") {
		t.Fatalf("expected stream error surfaced, got %v", err)
	}
}

func TestStreamTTSHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	_, err := c.StreamTTS(context.Background(), "v1", tts.TTSRequest{Text: "hi"}, 0)
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("expected 400 error, got %v", err)
	}
}

func TestToSixtyDBFormat(t *testing.T) {
	cases := map[string]string{
		"mp3_44100_128": "mp3",
		"pcm_44100":     "wav",
		"wav":           "wav",
		"opus_48000_64": "ogg",
		"ogg":           "ogg",
		"flac":          "flac",
		"":              "",
	}
	for in, want := range cases {
		if got := toSixtyDBFormat(in); got != want {
			t.Fatalf("toSixtyDBFormat(%q) = %q, want %q", in, got, want)
		}
	}
}
