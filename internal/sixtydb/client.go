// Package sixtydb provides a small client for the 60db (api.60db.ai) TTS API.
//
// Unlike ElevenLabs, 60db never returns raw audio: the synthesize endpoint
// wraps it as base64 inside a JSON object, and the stream endpoint emits
// newline-delimited JSON frames whose audio is base64 too. This client decodes
// both so StreamTTS/ConvertTTS hand back plain audio bytes, exactly like the
// ElevenLabs client — keeping the command and audio layers provider-agnostic.
package sixtydb

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/steipete/sag/internal/tts"
)

// DefaultBaseURL is the public 60db API host.
const DefaultBaseURL = "https://api.60db.ai"

// Client talks to the 60db HTTP API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Ensure the 60db client satisfies the shared provider contract.
var _ tts.Provider = (*Client)(nil)

// NewClient returns a Client configured with the given API key and base URL.
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) newRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, endpoint)

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	return req, nil
}

// --- Voices ---------------------------------------------------------------

type voiceEntry struct {
	VoiceID     string            `json:"voice_id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Model       string            `json:"model"`
	Labels      map[string]string `json:"labels"`
	Description *string           `json:"description"`
}

type myVoicesResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    []voiceEntry `json:"data"`
}

func (e voiceEntry) toVoice() tts.Voice {
	labels := make(map[string]string, len(e.Labels)+1)
	for k, v := range e.Labels {
		labels[k] = v
	}
	// Fold the model name into labels so `--label model=...` and query ranking
	// can match on it, mirroring how ElevenLabs exposes metadata via labels.
	if e.Model != "" {
		if _, ok := labels["model"]; !ok {
			labels["model"] = e.Model
		}
	}
	desc := ""
	if e.Description != nil {
		desc = *e.Description
	}
	return tts.Voice{
		VoiceID:     e.VoiceID,
		Name:        e.Name,
		Category:    e.Category,
		Description: desc,
		Labels:      labels,
		// 60db /myvoices exposes no preview URL, so previews are unavailable.
		PreviewURL: "",
	}
}

// ListVoices fetches the caller's available 60db voices.
func (c *Client) ListVoices(ctx context.Context) ([]tts.Voice, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/myvoices", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list voices failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var body myVoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	voices := make([]tts.Voice, 0, len(body.Data))
	for _, e := range body.Data {
		voices = append(voices, e.toVoice())
	}
	return voices, nil
}

// SearchVoices filters the voice list by a case-insensitive name substring.
// 60db has no server-side search, so this is done client-side.
func (c *Client) SearchVoices(ctx context.Context, search string, limit int) ([]tts.Voice, error) {
	voices, err := c.ListVoices(ctx)
	if err != nil {
		return nil, err
	}
	search = strings.TrimSpace(search)
	if search != "" {
		searchLower := strings.ToLower(search)
		filtered := make([]tts.Voice, 0, len(voices))
		for _, v := range voices {
			if strings.Contains(strings.ToLower(v.Name), searchLower) {
				filtered = append(filtered, v)
			}
		}
		voices = filtered
	}
	if limit > 0 && len(voices) > limit {
		voices = voices[:limit]
	}
	return voices, nil
}

// GetVoice returns metadata for a specific voice. 60db has no single-voice
// endpoint, so it resolves against the full list.
func (c *Client) GetVoice(ctx context.Context, voiceID string) (tts.Voice, error) {
	voices, err := c.ListVoices(ctx)
	if err != nil {
		return tts.Voice{}, err
	}
	for _, v := range voices {
		if v.VoiceID == voiceID {
			return v, nil
		}
	}
	return tts.Voice{}, fmt.Errorf("voice %q not found", voiceID)
}

// --- Text to speech -------------------------------------------------------

type synthesizeRequest struct {
	Text         string   `json:"text"`
	VoiceID      string   `json:"voice_id,omitempty"`
	Speed        *float64 `json:"speed,omitempty"`
	Stability    *float64 `json:"stability,omitempty"`
	Similarity   *float64 `json:"similarity,omitempty"`
	OutputFormat string   `json:"output_format,omitempty"`
}

// buildBody translates a provider-neutral request into 60db's payload.
// includeFormat is false for the streaming endpoint, whose spec omits
// output_format. Fields with no 60db equivalent (model, style, speaker boost,
// seed, normalization, language) are intentionally dropped.
func buildBody(voiceID string, req tts.TTSRequest, includeFormat bool) synthesizeRequest {
	body := synthesizeRequest{
		Text:    req.Text,
		VoiceID: voiceID,
	}
	if vs := req.VoiceSettings; vs != nil {
		if vs.Speed != nil {
			body.Speed = vs.Speed // both APIs use 0.5..2.0
		}
		if vs.Stability != nil {
			body.Stability = scaleToHundred(*vs.Stability)
		}
		if vs.SimilarityBoost != nil {
			body.Similarity = scaleToHundred(*vs.SimilarityBoost)
		}
	}
	if includeFormat {
		body.OutputFormat = toSixtyDBFormat(req.OutputFormat)
	}
	return body
}

// scaleToHundred converts a 0..1 knob to 60db's 0..100 scale.
func scaleToHundred(v float64) *float64 {
	scaled := v * 100
	return &scaled
}

// toSixtyDBFormat maps ElevenLabs-style format strings to 60db's simple set
// (mp3|wav|ogg|flac). Empty input yields empty (provider default).
func toSixtyDBFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	switch {
	case format == "":
		return ""
	case strings.HasPrefix(format, "mp3"):
		return "mp3"
	case strings.HasPrefix(format, "pcm"), strings.HasPrefix(format, "wav"):
		return "wav"
	case strings.HasPrefix(format, "opus"), strings.HasPrefix(format, "ogg"):
		return "ogg"
	case strings.HasPrefix(format, "flac"):
		return "flac"
	default:
		return format
	}
}

type synthesizeResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	AudioBase64 string `json:"audio_base64"`
}

// ConvertTTS downloads the full audio and returns decoded bytes.
func (c *Client) ConvertTTS(ctx context.Context, voiceID string, payload tts.TTSRequest) ([]byte, error) {
	bodyBytes, err := json.Marshal(buildBody(voiceID, payload, true))
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/tts-synthesize", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("convert TTS failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var body synthesizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if !body.Success && body.Message != "" {
		return nil, fmt.Errorf("convert TTS failed: %s", body.Message)
	}
	data, err := base64.StdEncoding.DecodeString(body.AudioBase64)
	if err != nil {
		return nil, fmt.Errorf("decode audio_base64: %w", err)
	}
	return data, nil
}

// StreamTTS requests streaming audio and returns a reader that yields decoded
// audio bytes. The latency argument is ignored (60db's stream has no tier).
func (c *Client) StreamTTS(ctx context.Context, voiceID string, payload tts.TTSRequest, _ int) (io.ReadCloser, error) {
	bodyBytes, err := json.Marshal(buildBody(voiceID, payload, false))
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/tts-stream", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stream TTS failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return newNDJSONAudioReader(resp.Body), nil
}

// --- NDJSON streaming decoder ---------------------------------------------

type streamFrame struct {
	Type   string `json:"type"`
	Result struct {
		AudioContent string `json:"audioContent"`
	} `json:"result"`
	Message string `json:"message"`
}

// ndjsonAudioReader unwraps 60db's newline-delimited JSON stream into a plain
// audio byte stream. Each "chunk" frame's base64 audio is decoded and served;
// "complete" ends the stream; "error" surfaces the message.
type ndjsonAudioReader struct {
	src     io.ReadCloser
	reader  *bufio.Reader
	pending []byte
	err     error
}

func newNDJSONAudioReader(src io.ReadCloser) *ndjsonAudioReader {
	return &ndjsonAudioReader{
		src:    src,
		reader: bufio.NewReader(src),
	}
}

func (r *ndjsonAudioReader) Read(p []byte) (int, error) {
	for len(r.pending) == 0 {
		if r.err != nil {
			return 0, r.err
		}
		if err := r.fill(); err != nil {
			r.err = err
			if len(r.pending) == 0 {
				return 0, err
			}
		}
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

// fill reads and decodes the next non-empty frame into r.pending.
func (r *ndjsonAudioReader) fill() error {
	for {
		line, err := r.reader.ReadBytes('\n')
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			var frame streamFrame
			if jerr := json.Unmarshal(trimmed, &frame); jerr != nil {
				return fmt.Errorf("decode stream frame: %w", jerr)
			}
			switch frame.Type {
			case "chunk":
				audio, derr := base64.StdEncoding.DecodeString(frame.Result.AudioContent)
				if derr != nil {
					return fmt.Errorf("decode audio chunk: %w", derr)
				}
				if len(audio) > 0 {
					r.pending = audio
					return nil
				}
			case "complete":
				return io.EOF
			case "error":
				if frame.Message != "" {
					return fmt.Errorf("stream error: %s", frame.Message)
				}
				return fmt.Errorf("stream error")
			}
			// Unknown frame types are ignored; keep reading.
		}
		if err != nil {
			if err == io.EOF {
				return io.EOF
			}
			return err
		}
	}
}

func (r *ndjsonAudioReader) Close() error {
	return r.src.Close()
}
