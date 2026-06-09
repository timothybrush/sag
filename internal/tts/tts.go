// Package tts defines provider-neutral types and the Provider interface shared
// by every text-to-speech backend (ElevenLabs, 60db, ...).
//
// Keeping the types here lets each provider implementation translate its own
// wire format to and from a single shared shape, so the command layer and the
// audio player never need to know which backend produced the audio.
package tts

import (
	"context"
	"io"
)

// Voice represents a single voice entry, normalized across providers.
type Voice struct {
	VoiceID     string            `json:"voice_id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels,omitempty"`
	PreviewURL  string            `json:"preview_url"`
}

// VoiceSettings tunes synthesis parameters for a request. All fields are
// pointers so unset knobs are omitted from the wire payload and the provider's
// own defaults apply. Stability/SimilarityBoost/Style use the 0..1 scale; each
// provider translates to its native range.
type VoiceSettings struct {
	Stability       *float64 `json:"stability,omitempty"`
	SimilarityBoost *float64 `json:"similarity_boost,omitempty"`
	Style           *float64 `json:"style,omitempty"`
	UseSpeakerBoost *bool    `json:"use_speaker_boost,omitempty"`
	Speed           *float64 `json:"speed,omitempty"`
}

// TTSRequest configures a text-to-speech request. Some fields are honored only
// by certain providers (e.g. ModelID/Seed/LanguageCode are ElevenLabs-specific
// and ignored by 60db); the provider implementation decides what to send.
type TTSRequest struct {
	Text                   string         `json:"text"`
	ModelID                string         `json:"model_id,omitempty"`
	VoiceSettings          *VoiceSettings `json:"voice_settings,omitempty"`
	OutputFormat           string         `json:"output_format,omitempty"`
	Seed                   *uint32        `json:"seed,omitempty"`
	ApplyTextNormalization string         `json:"apply_text_normalization,omitempty"`
	LanguageCode           string         `json:"language_code,omitempty"`
}

// Provider is the contract every TTS backend implements. StreamTTS and
// ConvertTTS must return raw, ready-to-play audio bytes (decoded/unwrapped from
// any provider-specific envelope) so the audio layer stays provider-agnostic.
type Provider interface {
	ListVoices(ctx context.Context) ([]Voice, error)
	SearchVoices(ctx context.Context, search string, limit int) ([]Voice, error)
	GetVoice(ctx context.Context, voiceID string) (Voice, error)
	StreamTTS(ctx context.Context, voiceID string, req TTSRequest, latency int) (io.ReadCloser, error)
	ConvertTTS(ctx context.Context, voiceID string, req TTSRequest) ([]byte, error)
}
