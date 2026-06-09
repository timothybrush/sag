package cmd

import (
	"fmt"
	"os"
	"strings"
)

// resolveElevenLabsKey resolves the ElevenLabs API key without erroring when
// absent (returns ""). Order: --api-key, key file, ELEVENLABS_API_KEY,
// SAG_API_KEY.
func resolveElevenLabsKey() (string, error) {
	if cfg.APIKey != "" {
		return cfg.APIKey, nil
	}
	key, err := resolveAPIKeyFromFile()
	if err != nil {
		return "", err
	}
	if key != "" {
		return key, nil
	}
	if v := os.Getenv("ELEVENLABS_API_KEY"); v != "" {
		return v, nil
	}
	if v := os.Getenv("SAG_API_KEY"); v != "" {
		return v, nil
	}
	return "", nil
}

// ensureAPIKey resolves and stores the ElevenLabs API key, erroring if missing.
func ensureAPIKey() error {
	key, err := resolveElevenLabsKey()
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("missing ElevenLabs API key (set --api-key, --api-key-file, or ELEVENLABS_API_KEY)")
	}
	cfg.APIKey = key
	return nil
}

func resolveAPIKeyFromFile() (string, error) {
	path := cfg.APIKeyFile
	if path == "" {
		path = os.Getenv("ELEVENLABS_API_KEY_FILE")
	}
	if path == "" {
		path = os.Getenv("SAG_API_KEY_FILE")
	}
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read api key file: %w", err)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("api key file %q is empty", path)
	}
	return key, nil
}
