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

// resolveSixtyDBKey resolves the 60db API key from its env vars.
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

// ensureProviderConfigured verifies that at least one provider's key is set.
// Used by PreRunE so 60db-only users aren't rejected for lacking an
// ElevenLabs key.
func ensureProviderConfigured() error {
	elKey, err := resolveElevenLabsKey()
	if err != nil {
		return err
	}
	sdKey, err := resolveSixtyDBKey()
	if err != nil {
		return err
	}
	if elKey == "" && sdKey == "" {
		return fmt.Errorf("missing API key (set ELEVENLABS_API_KEY or SIXTYDB_API_KEY; or --api-key / --api-key-file)")
	}
	return nil
}

// selectProvider auto-detects the active provider from whichever API key is
// present. If both are set, ElevenLabs wins (preserving prior default) and a
// note is printed. The chosen client is built with cfg.BaseURL, which each
// client treats as a per-provider override (empty => provider default host).
func selectProvider() (tts.Provider, string, error) {
	elKey, err := resolveElevenLabsKey()
	if err != nil {
		return nil, "", err
	}
	sdKey, err := resolveSixtyDBKey()
	if err != nil {
		return nil, "", err
	}

	switch {
	case elKey != "" && sdKey != "":
		fmt.Fprintln(os.Stderr, "note: both ElevenLabs and 60db API keys set; using ElevenLabs (unset ELEVENLABS_API_KEY to use 60db)")
		return elevenlabs.NewClient(elKey, cfg.BaseURL), providerElevenLabs, nil
	case elKey != "":
		return elevenlabs.NewClient(elKey, cfg.BaseURL), providerElevenLabs, nil
	case sdKey != "":
		return sixtydb.NewClient(sdKey, cfg.BaseURL), providerSixtyDB, nil
	default:
		return nil, "", fmt.Errorf("missing API key (set ELEVENLABS_API_KEY or SIXTYDB_API_KEY; or --api-key / --api-key-file)")
	}
}

// sixtyDBOnlyFlags lists flags that ElevenLabs honors but 60db has no
// equivalent for. When the active provider is 60db and the user set one, we
// note that it is ignored rather than failing.
var sixtyDBIgnoredFlags = []string{
	"model-id", "style", "speaker-boost", "no-speaker-boost",
	"seed", "normalize", "lang", "latency-tier",
}

// noteUnsupportedSixtyDBFlags prints a single stderr note if the user set any
// flag that 60db ignores.
func noteUnsupportedSixtyDBFlags(changed func(string) bool) {
	var ignored []string
	for _, name := range sixtyDBIgnoredFlags {
		if changed(name) {
			ignored = append(ignored, "--"+name)
		}
	}
	if len(ignored) > 0 {
		fmt.Fprintf(os.Stderr, "note: 60db ignores %s\n", strings.Join(ignored, ", "))
	}
}
