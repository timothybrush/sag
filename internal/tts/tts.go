// Package tts holds the small voice-catalog types shared by the existing
// `voices` and voice-resolution command paths.
package tts

import (
	"context"
)

// Voice represents a single voice entry normalized for the CLI's voice listing,
// filtering, query, and resolution paths.
type Voice struct {
	VoiceID     string            `json:"voice_id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels,omitempty"`
	PreviewURL  string            `json:"preview_url"`
}

// VoiceCatalog is the minimal shared interface needed by existing commands.
type VoiceCatalog interface {
	ListVoices(ctx context.Context) ([]Voice, error)
	SearchVoices(ctx context.Context, search string, limit int) ([]Voice, error)
	GetVoice(ctx context.Context, voiceID string) (Voice, error)
}
