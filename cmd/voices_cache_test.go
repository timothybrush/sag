package cmd

import (
	"testing"

	"github.com/steipete/sag/internal/tts"
)

func TestMergeVoicePreservesAndOverlaysLabels(t *testing.T) {
	base := tts.Voice{
		VoiceID: "v1",
		Labels: map[string]string{
			"source": "default",
			"model":  "60db Quality",
		},
	}
	details := tts.Voice{
		VoiceID: "v1",
		Labels: map[string]string{
			"accent": "American",
			"model":  "override",
		},
		PreviewURL: "https://cdn.example.com/sample.mp3",
	}

	merged := mergeVoice(base, details)
	if merged.Labels["source"] != "default" {
		t.Fatalf("expected source label preserved, got %+v", merged.Labels)
	}
	if merged.Labels["model"] != "override" || merged.Labels["accent"] != "American" {
		t.Fatalf("expected detail labels merged, got %+v", merged.Labels)
	}
	if merged.PreviewURL != details.PreviewURL {
		t.Fatalf("expected preview URL copied, got %q", merged.PreviewURL)
	}
}
