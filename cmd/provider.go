package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/steipete/sag/internal/elevenlabs"
	"github.com/steipete/sag/internal/sixtydb"
	"github.com/steipete/sag/internal/tts"
)

const (
	providerElevenLabs = "elevenlabs"
	providerSixtyDB    = "60db"
)

type activeProvider struct {
	name   string
	voices tts.VoiceCatalog

	elevenlabs *elevenlabs.Client
	sixtydb    *sixtydb.Client
}

// resolveSixtyDBKey resolves the 60db API key from its dedicated env vars.
// Order: SIXTYDB_API_KEY, then SIXTYDB_API_KEY_FILE.
func resolveSixtyDBKey() (string, error) {
	if key := strings.TrimSpace(os.Getenv("SIXTYDB_API_KEY")); key != "" {
		return key, nil
	}
	if path := strings.TrimSpace(os.Getenv("SIXTYDB_API_KEY_FILE")); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read 60db api key file: %w", err)
		}
		key := strings.TrimSpace(string(data))
		if key == "" {
			return "", fmt.Errorf("60db api key file %q is empty", path)
		}
		return key, nil
	}
	return "", nil
}

func ensureProviderConfigured() error {
	_, err := selectProvider()
	return err
}

func selectProvider() (activeProvider, error) {
	elKey, err := resolveElevenLabsKey()
	if err != nil {
		return activeProvider{}, err
	}
	sdKey, err := resolveSixtyDBKey()
	if err != nil {
		return activeProvider{}, err
	}

	switch {
	case elKey != "" && sdKey != "":
		return activeProvider{}, fmt.Errorf("ambiguous provider configuration: both ElevenLabs and 60db keys are set; unset one provider key and retry")
	case elKey != "":
		client := elevenlabs.NewClient(elKey, cfg.BaseURL)
		return activeProvider{
			name:       providerElevenLabs,
			voices:     client,
			elevenlabs: client,
		}, nil
	case sdKey != "":
		client := sixtydb.NewClient(sdKey, cfg.BaseURL)
		return activeProvider{
			name:    providerSixtyDB,
			voices:  client,
			sixtydb: client,
		}, nil
	default:
		return activeProvider{}, fmt.Errorf("missing API key (set ELEVENLABS_API_KEY or SIXTYDB_API_KEY)")
	}
}

var sixtyDBUnsupportedFlags = []string{
	"model-id",
	"style",
	"speaker-boost",
	"no-speaker-boost",
	"seed",
	"normalize",
	"lang",
	"latency-tier",
}

func changedSixtyDBUnsupportedFlags(changed func(string) bool) []string {
	var unsupported []string
	for _, name := range sixtyDBUnsupportedFlags {
		if changed(name) {
			unsupported = append(unsupported, "--"+name)
		}
	}
	return unsupported
}
