package sixtydb

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientDefaultsBase(t *testing.T) {
	c := NewClient("key", "")
	if c.baseURL != DefaultBaseURL {
		t.Fatalf("unexpected baseURL: %s", c.baseURL)
	}
}

func TestListVoicesMergesDefaultAndMyVoices(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		switch r.URL.Path {
		case "/default-voices":
			_, _ = io.WriteString(w, `{"success":true,"data":[
				{"voice_id":"v1","name":"Aria","category":"default","model":"60db Quality","labels":{"accent":"US"}},
				{"voice_id":"dup","name":"Default Dup","category":"default"}
			]}`)
		case "/myvoices":
			_, _ = io.WriteString(w, `{"success":true,"data":[
				{"voice_id":"dup","name":"My Dup","category":"cloned"},
				{"voice_id":"v2","name":"Ravi","category":"cloned","categories":["narration","warm"]}
			]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	voices, err := c.ListVoices(context.Background())
	if err != nil {
		t.Fatalf("ListVoices error: %v", err)
	}
	if want := []string{"/default-voices", "/myvoices"}; strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Fatalf("route order = %v, want %v", calls, want)
	}
	if len(voices) != 3 {
		t.Fatalf("expected 3 merged voices, got %d", len(voices))
	}
	if voices[0].VoiceID != "v1" || voices[0].Labels["source"] != "default" || voices[0].Labels["model"] != "60db Quality" {
		t.Fatalf("unexpected default voice: %+v", voices[0])
	}
	if voices[1].VoiceID != "dup" || voices[1].Name != "Default Dup" {
		t.Fatalf("expected default duplicate to win, got %+v", voices[1])
	}
	if voices[2].VoiceID != "v2" || voices[2].Labels["source"] != "myvoices" || voices[2].Labels["categories"] != "narration, warm" {
		t.Fatalf("unexpected user voice: %+v", voices[2])
	}
}

func TestSearchVoicesFiltersAndLimits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/default-voices":
			_, _ = io.WriteString(w, `{"success":true,"data":[
				{"voice_id":"v1","name":"Roger"},
				{"voice_id":"v2","name":"Rogue"}
			]}`)
		case "/myvoices":
			_, _ = io.WriteString(w, `{"success":true,"data":[{"voice_id":"v3","name":"Sarah"}]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	voices, err := c.SearchVoices(context.Background(), "rog", 1)
	if err != nil {
		t.Fatalf("SearchVoices error: %v", err)
	}
	if len(voices) != 1 || voices[0].VoiceID != "v1" {
		t.Fatalf("unexpected voices: %+v", voices)
	}
}

func TestGetVoiceUsesDocumentedPerVoiceRoute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/voices/voice-001" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{
			"id":"voice-001",
			"name":"Sarah",
			"description":"Professional female voice",
			"language":"en-US",
			"gender":"female",
			"age":"middle",
			"accent":"American",
			"use_case":["narration","customer-service"],
			"sample_url":"https://cdn.60db.com/samples/voice-001.mp3",
			"is_custom":false
		}`)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	voice, err := c.GetVoice(context.Background(), "voice-001")
	if err != nil {
		t.Fatalf("GetVoice error: %v", err)
	}
	if voice.VoiceID != "voice-001" || voice.Name != "Sarah" {
		t.Fatalf("unexpected voice: %+v", voice)
	}
	if voice.PreviewURL != "https://cdn.60db.com/samples/voice-001.mp3" {
		t.Fatalf("unexpected preview URL: %q", voice.PreviewURL)
	}
	if voice.Labels["use_case"] != "narration, customer-service" || voice.Labels["accent"] != "American" {
		t.Fatalf("unexpected voice labels: %+v", voice.Labels)
	}
}

func TestListVoicesRejectsHTTP200ErrorEnvelopeWithoutLeakingToken(t *testing.T) {
	const secret = "secret-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"message":"Bearer `+secret+` invalid token"}`)
	}))
	defer srv.Close()

	c := NewClient(secret, srv.URL)
	_, err := c.ListVoices(context.Background())
	assertSanitizedError(t, err, secret, "invalid token")
}

func TestConvertTTSUsesDocumentedRouteAndValidatesResponse(t *testing.T) {
	audio := []byte("ID3converted-audio")
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
		if body["text"] != "hi" || body["voice_id"] != "v1" {
			t.Fatalf("unexpected request body: %+v", body)
		}
		if body["speed"] != 1.1 || body["stability"] != 50.0 || body["similarity"] != 80.0 || body["output_format"] != "mp3" {
			t.Fatalf("unexpected mapped request body: %+v", body)
		}

		resp := map[string]any{
			"success":       true,
			"audio_base64":  base64.StdEncoding.EncodeToString(audio),
			"encoding":      "mp3",
			"output_format": "mp3",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	speed := 1.1
	stability := 50.0
	similarity := 80.0
	c := NewClient("key", srv.URL)
	got, err := c.ConvertTTS(context.Background(), TTSRequest{
		Text:         "hi",
		VoiceID:      "v1",
		Speed:        &speed,
		Stability:    &stability,
		Similarity:   &similarity,
		OutputFormat: "mp3_44100_128",
	})
	if err != nil {
		t.Fatalf("ConvertTTS error: %v", err)
	}
	if !bytes.Equal(got, audio) {
		t.Fatalf("decoded audio = %q, want %q", got, audio)
	}
}

func TestConvertTTSRejectsHTTP200ErrorEnvelopeWithoutLeakingToken(t *testing.T) {
	const secret = "secret-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"message":"Bearer `+secret+` invalid token"}`)
	}))
	defer srv.Close()

	c := NewClient(secret, srv.URL)
	_, err := c.ConvertTTS(context.Background(), TTSRequest{Text: "hi"})
	assertSanitizedError(t, err, secret, "invalid token")
}

func TestConvertTTSRejectsMalformedOrMismatchedAudio(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "empty audio",
			body: `{"success":true,"audio_base64":""}`,
			want: "empty audio_base64",
		},
		{
			name: "unknown format",
			body: `{"success":true,"audio_base64":"` + mustBase64([]byte("not-audio")) + `"}`,
			want: "unrecognized audio format",
		},
		{
			name: "declared format mismatch",
			body: `{"success":true,"audio_base64":"` + mustBase64([]byte("ID3mp3-data")) + `","output_format":"wav"}`,
			want: "declared wav, decoded mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			c := NewClient("key", srv.URL)
			_, err := c.ConvertTTS(context.Background(), TTSRequest{Text: "hi"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestStreamTTSUsesDocumentedRouteAndDecodesNDJSON(t *testing.T) {
	chunk1 := mustBase64([]byte("ID3hello-"))
	chunk2 := mustBase64([]byte("world"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tts-stream" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := body["output_format"]; ok {
			t.Fatalf("expected output_format omitted from stream body")
		}
		_, _ = io.WriteString(w, `{"type":"chunk","result":{"audioContent":"`+chunk1+`"}}`+"\n")
		_, _ = io.WriteString(w, `{"type":"chunk","result":{"audioContent":"`+chunk2+`"}}`+"\n")
		_, _ = io.WriteString(w, `{"type":"complete"}`+"\n")
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	rc, err := c.StreamTTS(context.Background(), TTSRequest{Text: "hi"})
	if err != nil {
		t.Fatalf("StreamTTS error: %v", err)
	}
	defer func() { _ = rc.Close() }()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if string(got) != "ID3hello-world" {
		t.Fatalf("unexpected stream body: %q", got)
	}
}

func TestStreamTTSRejectsInvalidTokenWithoutLeakingToken(t *testing.T) {
	const secret = "secret-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"success":false,"message":"Bearer `+secret+` invalid token"}`)
	}))
	defer srv.Close()

	c := NewClient(secret, srv.URL)
	_, err := c.StreamTTS(context.Background(), TTSRequest{Text: "hi"})
	assertSanitizedError(t, err, secret, "invalid token")
}

func TestStreamTTSRejectsMalformedStreams(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  string
		isErr error
	}{
		{
			name:  "bad json",
			lines: []string{`not-json`},
			want:  "decode stream frame",
		},
		{
			name:  "unknown frame type",
			lines: []string{`{"type":"wat"}`},
			want:  `unknown stream frame type "wat"`,
		},
		{
			name:  "missing audio content",
			lines: []string{`{"type":"chunk","result":{"audioContent":""}}`},
			want:  "missing audioContent",
		},
		{
			name:  "unknown audio format",
			lines: []string{`{"type":"chunk","result":{"audioContent":"` + mustBase64([]byte("bad")) + `"}}`},
			want:  "unknown streamed audio format",
		},
		{
			name:  "complete without audio",
			lines: []string{`{"type":"complete"}`},
			want:  "stream completed without audio",
		},
		{
			name:  "empty stream",
			lines: nil,
			isErr: io.ErrUnexpectedEOF,
		},
		{
			name:  "missing complete frame",
			lines: []string{`{"type":"chunk","result":{"audioContent":"` + mustBase64([]byte("ID3chunk")) + `"}}`},
			isErr: io.ErrUnexpectedEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := newNDJSONAudioReader(context.Background(), io.NopCloser(strings.NewReader(strings.Join(tt.lines, "\n"))), nil)
			defer func() { _ = reader.Close() }()

			_, err := io.ReadAll(reader)
			if tt.isErr != nil {
				if !errors.Is(err, tt.isErr) {
					t.Fatalf("expected %v, got %v", tt.isErr, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestNDJSONAudioReaderHonorsCancellation(t *testing.T) {
	pr, pw := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	reader := newNDJSONAudioReader(ctx, pr, nil)
	defer func() { _ = reader.Close() }()

	done := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(reader)
		done <- err
	}()

	cancel()
	_ = pw.Close()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stream read did not unblock on cancellation")
	}
}

func TestNDJSONAudioReaderRejectsOversizedFrame(t *testing.T) {
	line := strings.Repeat("a", maxStreamFrameBytes+1)
	reader := newNDJSONAudioReader(context.Background(), io.NopCloser(strings.NewReader(line)), nil)
	defer func() { _ = reader.Close() }()

	_, err := io.ReadAll(reader)
	if err == nil || !strings.Contains(err.Error(), "token too long") {
		t.Fatalf("expected oversized frame error, got %v", err)
	}
}

func TestNDJSONAudioReaderSanitizesErrorFrames(t *testing.T) {
	const secret = "secret-token"
	reader := newNDJSONAudioReader(
		context.Background(),
		io.NopCloser(strings.NewReader(`{"type":"error","message":"Bearer `+secret+` invalid token"}`)),
		func(msg string) string {
			return strings.ReplaceAll(msg, secret, "[redacted]")
		},
	)
	defer func() { _ = reader.Close() }()

	_, err := io.ReadAll(reader)
	assertSanitizedError(t, err, secret, "invalid token")
}

func TestDecodeChunkRejectsPerChunkAndTotalLimits(t *testing.T) {
	tooLargeChunk := mustBase64(bytes.Repeat([]byte{'a'}, maxDecodedChunkBytes+1))
	if _, err := decodeChunk(tooLargeChunk, 0); err == nil || !strings.Contains(err.Error(), "stream chunk exceeds") {
		t.Fatalf("expected chunk limit error, got %v", err)
	}

	if _, err := decodeChunk(mustBase64([]byte("abc")), maxDecodedAudioBytes-2); err == nil || !strings.Contains(err.Error(), "stream audio exceeds") {
		t.Fatalf("expected total limit error, got %v", err)
	}
}

func TestCanonicalOutputFormat(t *testing.T) {
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
		if got := CanonicalOutputFormat(in); got != want {
			t.Fatalf("CanonicalOutputFormat(%q) = %q, want %q", in, got, want)
		}
	}
}

func mustBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func assertSanitizedError(t *testing.T, err error, secret, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked secret: %v", err)
	}
	if !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("expected redacted marker, got %v", err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q in error, got %v", want, err)
	}
}
