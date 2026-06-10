// Package sixtydb provides a strict adapter for the documented 60db HTTP TTS
// endpoints.
package sixtydb

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

const (
	maxDecodedAudioBytes = 96 << 20
	maxDecodedChunkBytes = 8 << 20
	maxStreamFrameBytes  = 12 << 20
)

type audioFormat string

const (
	audioFormatMP3  audioFormat = "mp3"
	audioFormatWAV  audioFormat = "wav"
	audioFormatOGG  audioFormat = "ogg"
	audioFormatFLAC audioFormat = "flac"
)

// Client talks to the 60db HTTP API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// TTSRequest is the documented 60db TTS payload shape exposed by the CLI.
type TTSRequest struct {
	Text         string   `json:"text"`
	VoiceID      string   `json:"voice_id,omitempty"`
	Speed        *float64 `json:"speed,omitempty"`
	Stability    *float64 `json:"stability,omitempty"`
	Similarity   *float64 `json:"similarity,omitempty"`
	OutputFormat string   `json:"output_format,omitempty"`
}

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

type envelope struct {
	Success *bool  `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

func (c *Client) errorFromEnvelope(op string, env envelope, fallback string) error {
	msg := strings.TrimSpace(env.Message)
	if msg == "" {
		msg = strings.TrimSpace(env.Error)
	}
	if msg == "" {
		msg = fallback
	}
	return fmt.Errorf("%s: %s", op, c.sanitize(msg))
}

func (c *Client) httpError(op string, status string, body []byte) error {
	var env envelope
	if len(body) > 0 && json.Unmarshal(body, &env) == nil {
		return c.errorFromEnvelope(op, env, status)
	}
	return fmt.Errorf("%s: %s", op, status)
}

func (c *Client) sanitize(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" || c.apiKey == "" {
		return msg
	}
	msg = strings.ReplaceAll(msg, "Bearer "+c.apiKey, "Bearer [redacted]")
	msg = strings.ReplaceAll(msg, c.apiKey, "[redacted]")
	return msg
}

// CanonicalOutputFormat maps CLI/ElevenLabs-style output names to the simple
// 60db formats documented for /tts-synthesize.
func CanonicalOutputFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	switch {
	case format == "":
		return ""
	case strings.HasPrefix(format, "mp3"):
		return string(audioFormatMP3)
	case strings.HasPrefix(format, "pcm"), strings.HasPrefix(format, "wav"):
		return string(audioFormatWAV)
	case strings.HasPrefix(format, "opus"), strings.HasPrefix(format, "ogg"):
		return string(audioFormatOGG)
	case strings.HasPrefix(format, "flac"):
		return string(audioFormatFLAC)
	default:
		return format
	}
}

func parseAudioFormat(value string) audioFormat {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(audioFormatMP3):
		return audioFormatMP3
	case string(audioFormatWAV):
		return audioFormatWAV
	case string(audioFormatOGG):
		return audioFormatOGG
	case string(audioFormatFLAC):
		return audioFormatFLAC
	default:
		return ""
	}
}

func sniffAudioFormat(data []byte) (audioFormat, error) {
	if len(data) == 0 {
		return "", errors.New("empty audio")
	}
	switch {
	case len(data) >= 3 && string(data[:3]) == "ID3":
		return audioFormatMP3, nil
	case len(data) >= 2 && data[0] == 0xff && data[1]&0xe0 == 0xe0:
		return audioFormatMP3, nil
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE":
		return audioFormatWAV, nil
	case len(data) >= 4 && string(data[:4]) == "OggS":
		return audioFormatOGG, nil
	case len(data) >= 4 && string(data[:4]) == "fLaC":
		return audioFormatFLAC, nil
	default:
		return "", errors.New("unrecognized audio format")
	}
}

type voiceEntry struct {
	VoiceID     string            `json:"voice_id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Model       string            `json:"model"`
	Labels      map[string]string `json:"labels"`
	Description *string           `json:"description"`
	Categories  []string          `json:"categories"`
}

type voicesResponse struct {
	envelope
	Data []voiceEntry `json:"data"`
}

type voiceDetailsResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Gender      string   `json:"gender"`
	Age         string   `json:"age"`
	Accent      string   `json:"accent"`
	UseCase     []string `json:"use_case"`
	SampleURL   string   `json:"sample_url"`
	IsCustom    bool     `json:"is_custom"`
}

func (e voiceEntry) toVoice(source string) tts.Voice {
	labels := make(map[string]string, len(e.Labels)+2)
	for k, v := range e.Labels {
		labels[k] = v
	}
	if e.Model != "" {
		labels["model"] = e.Model
	}
	if len(e.Categories) > 0 {
		labels["categories"] = strings.Join(e.Categories, ", ")
	}
	if source != "" {
		labels["source"] = source
	}
	description := ""
	if e.Description != nil {
		description = strings.TrimSpace(*e.Description)
	}
	return tts.Voice{
		VoiceID:     e.VoiceID,
		Name:        e.Name,
		Category:    e.Category,
		Description: description,
		Labels:      labels,
	}
}

func (v voiceDetailsResponse) toVoice() tts.Voice {
	labels := map[string]string{}
	if v.Language != "" {
		labels["language"] = v.Language
	}
	if v.Gender != "" {
		labels["gender"] = v.Gender
	}
	if v.Age != "" {
		labels["age"] = v.Age
	}
	if v.Accent != "" {
		labels["accent"] = v.Accent
	}
	if len(v.UseCase) > 0 {
		labels["use_case"] = strings.Join(v.UseCase, ", ")
	}
	if v.IsCustom {
		labels["source"] = "myvoices"
	}
	return tts.Voice{
		VoiceID:     v.ID,
		Name:        v.Name,
		Description: strings.TrimSpace(v.Description),
		Labels:      labels,
		PreviewURL:  strings.TrimSpace(v.SampleURL),
	}
}

func (c *Client) fetchVoices(ctx context.Context, endpoint, source string) ([]tts.Voice, error) {
	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
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
		body, _ := io.ReadAll(resp.Body)
		return nil, c.httpError("list voices", resp.Status, body)
	}

	var body voicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("list voices: decode response: %w", err)
	}
	if body.Success == nil || !*body.Success {
		return nil, c.errorFromEnvelope("list voices", body.envelope, "unexpected response envelope")
	}

	voices := make([]tts.Voice, 0, len(body.Data))
	for _, entry := range body.Data {
		voices = append(voices, entry.toVoice(source))
	}
	return voices, nil
}

// ListVoices merges the documented default and user-created voice catalogs.
func (c *Client) ListVoices(ctx context.Context) ([]tts.Voice, error) {
	defaultVoices, err := c.fetchVoices(ctx, "/default-voices", "default")
	if err != nil {
		return nil, err
	}
	myVoices, err := c.fetchVoices(ctx, "/myvoices", "myvoices")
	if err != nil {
		return nil, err
	}

	merged := make([]tts.Voice, 0, len(defaultVoices)+len(myVoices))
	seen := make(map[string]struct{}, len(defaultVoices)+len(myVoices))
	for _, voice := range append(defaultVoices, myVoices...) {
		if voice.VoiceID == "" {
			continue
		}
		if _, ok := seen[voice.VoiceID]; ok {
			continue
		}
		seen[voice.VoiceID] = struct{}{}
		merged = append(merged, voice)
	}
	return merged, nil
}

// SearchVoices filters the merged voice catalog by name.
func (c *Client) SearchVoices(ctx context.Context, search string, limit int) ([]tts.Voice, error) {
	voices, err := c.ListVoices(ctx)
	if err != nil {
		return nil, err
	}
	search = strings.TrimSpace(search)
	if search != "" {
		searchLower := strings.ToLower(search)
		filtered := make([]tts.Voice, 0, len(voices))
		for _, voice := range voices {
			if strings.Contains(strings.ToLower(voice.Name), searchLower) {
				filtered = append(filtered, voice)
			}
		}
		voices = filtered
	}
	if limit > 0 && len(voices) > limit {
		voices = voices[:limit]
	}
	return voices, nil
}

// GetVoice resolves a voice from the documented per-voice endpoint.
func (c *Client) GetVoice(ctx context.Context, voiceID string) (tts.Voice, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("/voices", voiceID), nil)
	if err != nil {
		return tts.Voice{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return tts.Voice{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return tts.Voice{}, c.httpError("get voice", resp.Status, body)
	}

	var body voiceDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return tts.Voice{}, fmt.Errorf("get voice: decode response: %w", err)
	}
	if strings.TrimSpace(body.ID) == "" {
		return tts.Voice{}, errors.New("get voice: missing voice id")
	}
	return body.toVoice(), nil
}

type synthResponse struct {
	envelope
	AudioBase64     string  `json:"audio_base64"`
	SampleRate      int     `json:"sample_rate"`
	DurationSeconds float64 `json:"duration_seconds"`
	Encoding        string  `json:"encoding"`
	OutputFormat    string  `json:"output_format"`
}

func validateRequestedFormat(requested string) error {
	if requested == "" {
		return nil
	}
	if parseAudioFormat(requested) == "" {
		return fmt.Errorf("unsupported 60db output format %q", requested)
	}
	return nil
}

func validateConvertResponseFormat(requested string, resp synthResponse, data []byte) error {
	sniffed, err := sniffAudioFormat(data)
	if err != nil {
		return err
	}

	declared := parseAudioFormat(resp.OutputFormat)
	if declared == "" {
		declared = parseAudioFormat(resp.Encoding)
	}
	if declared != "" && declared != sniffed {
		return fmt.Errorf("response format mismatch: declared %s, decoded %s", declared, sniffed)
	}

	expected := parseAudioFormat(requested)
	if expected != "" && sniffed != expected {
		return fmt.Errorf("response format mismatch: requested %s, decoded %s", expected, sniffed)
	}
	return nil
}

// ConvertTTS downloads the full audio and returns decoded bytes.
func (c *Client) ConvertTTS(ctx context.Context, reqBody TTSRequest) ([]byte, error) {
	reqBody.OutputFormat = CanonicalOutputFormat(reqBody.OutputFormat)
	if err := validateRequestedFormat(reqBody.OutputFormat); err != nil {
		return nil, err
	}

	bodyBytes, err := json.Marshal(reqBody)
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
		body, _ := io.ReadAll(resp.Body)
		return nil, c.httpError("synthesize audio", resp.Status, body)
	}

	var body synthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("synthesize audio: decode response: %w", err)
	}
	if body.Success == nil || !*body.Success {
		return nil, c.errorFromEnvelope("synthesize audio", body.envelope, "unexpected response envelope")
	}
	if strings.TrimSpace(body.AudioBase64) == "" {
		return nil, errors.New("synthesize audio: empty audio_base64")
	}

	decodedLen := base64.StdEncoding.DecodedLen(len(body.AudioBase64))
	if decodedLen <= 0 {
		return nil, errors.New("synthesize audio: empty decoded audio")
	}
	if decodedLen > maxDecodedAudioBytes {
		return nil, fmt.Errorf("synthesize audio: decoded audio exceeds %d bytes", maxDecodedAudioBytes)
	}

	data := make([]byte, decodedLen)
	n, err := base64.StdEncoding.Decode(data, []byte(body.AudioBase64))
	if err != nil {
		return nil, fmt.Errorf("synthesize audio: decode audio_base64: %w", err)
	}
	data = data[:n]
	if len(data) == 0 {
		return nil, errors.New("synthesize audio: decoded audio was empty")
	}
	if err := validateConvertResponseFormat(reqBody.OutputFormat, body, data); err != nil {
		return nil, fmt.Errorf("synthesize audio: %w", err)
	}
	return data, nil
}

// StreamTTS requests streaming audio. The documented stream API does not
// accept output_format.
func (c *Client) StreamTTS(ctx context.Context, reqBody TTSRequest) (io.ReadCloser, error) {
	if CanonicalOutputFormat(reqBody.OutputFormat) != "" {
		return nil, errors.New("stream audio: output_format is not supported by /tts-stream")
	}

	bodyBytes, err := json.Marshal(reqBody)
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
		body, _ := io.ReadAll(resp.Body)
		return nil, c.httpError("stream audio", resp.Status, body)
	}
	return newNDJSONAudioReader(ctx, resp.Body, c.sanitize), nil
}

type streamFrame struct {
	Type   string `json:"type"`
	Result struct {
		AudioContent string `json:"audioContent"`
	} `json:"result"`
	Message string `json:"message"`
}

type ndjsonAudioReader struct {
	ctx      context.Context
	src      io.ReadCloser
	scanner  *bufio.Scanner
	pending  []byte
	err      error
	stop     func() bool
	sanitize func(string) string

	totalBytes  int64
	sawChunk    bool
	sawComplete bool
}

func newNDJSONAudioReader(ctx context.Context, src io.ReadCloser, sanitize func(string) string) *ndjsonAudioReader {
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 64*1024), maxStreamFrameBytes)
	if sanitize == nil {
		sanitize = func(msg string) string { return msg }
	}

	reader := &ndjsonAudioReader{
		ctx:      ctx,
		src:      src,
		scanner:  scanner,
		sanitize: sanitize,
	}
	reader.stop = context.AfterFunc(ctx, func() {
		_ = src.Close()
	})
	return reader
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

func (r *ndjsonAudioReader) fill() error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	for r.scanner.Scan() {
		line := bytes.TrimSpace(r.scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var frame streamFrame
		if err := json.Unmarshal(line, &frame); err != nil {
			return fmt.Errorf("decode stream frame: %w", err)
		}

		switch frame.Type {
		case "chunk":
			audio, err := decodeChunk(frame.Result.AudioContent, r.totalBytes)
			if err != nil {
				return err
			}
			if !r.sawChunk {
				if _, err := sniffAudioFormat(audio); err != nil {
					return fmt.Errorf("unknown streamed audio format: %w", err)
				}
			}
			r.sawChunk = true
			r.totalBytes += int64(len(audio))
			r.pending = audio
			return nil
		case "complete":
			r.sawComplete = true
			if !r.sawChunk {
				return errors.New("stream completed without audio")
			}
			return io.EOF
		case "error":
			if msg := strings.TrimSpace(frame.Message); msg != "" {
				return fmt.Errorf("stream error: %s", r.sanitize(msg))
			}
			return errors.New("stream error")
		default:
			return fmt.Errorf("unknown stream frame type %q", frame.Type)
		}
	}

	if err := r.scanner.Err(); err != nil {
		if ctxErr := r.ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("read stream frame: %w", err)
	}
	if ctxErr := r.ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if r.sawComplete {
		return io.EOF
	}
	return io.ErrUnexpectedEOF
}

func decodeChunk(encoded string, totalBytes int64) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, errors.New("stream chunk missing audioContent")
	}
	decodedLen := base64.StdEncoding.DecodedLen(len(encoded))
	if decodedLen <= 0 {
		return nil, errors.New("stream chunk decoded to empty audio")
	}
	if decodedLen > maxDecodedChunkBytes {
		return nil, fmt.Errorf("stream chunk exceeds %d bytes", maxDecodedChunkBytes)
	}
	if totalBytes+int64(decodedLen) > maxDecodedAudioBytes {
		return nil, fmt.Errorf("stream audio exceeds %d bytes", maxDecodedAudioBytes)
	}

	audio := make([]byte, decodedLen)
	n, err := base64.StdEncoding.Decode(audio, []byte(encoded))
	if err != nil {
		return nil, fmt.Errorf("decode audio chunk: %w", err)
	}
	audio = audio[:n]
	if len(audio) == 0 {
		return nil, errors.New("stream chunk decoded to empty audio")
	}
	return audio, nil
}

func (r *ndjsonAudioReader) Close() error {
	if r.stop != nil {
		r.stop()
	}
	return r.src.Close()
}
