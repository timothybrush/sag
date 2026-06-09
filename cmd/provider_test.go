package cmd

import (
	"testing"
)

// resetProviderEnv neutralizes every key source so each case starts clean.
// Setting an env var to "" makes the resolvers treat it as absent.
func resetProviderEnv(t *testing.T) {
	t.Helper()
	cfg.APIKey = ""
	cfg.APIKeyFile = ""
	cfg.BaseURL = ""
	t.Cleanup(func() { cfg.APIKey = ""; cfg.APIKeyFile = ""; cfg.BaseURL = "" })
	for _, k := range []string{
		"ELEVENLABS_API_KEY", "SAG_API_KEY",
		"ELEVENLABS_API_KEY_FILE", "SAG_API_KEY_FILE",
		"SIXTYDB_API_KEY", "SIXTYDB_API_KEY_FILE",
	} {
		t.Setenv(k, "")
	}
}

func TestSelectProvider_ElevenLabsOnly(t *testing.T) {
	resetProviderEnv(t)
	t.Setenv("ELEVENLABS_API_KEY", "el-key")

	_, name, err := selectProvider()
	if err != nil {
		t.Fatalf("selectProvider error: %v", err)
	}
	if name != providerElevenLabs {
		t.Fatalf("expected %q, got %q", providerElevenLabs, name)
	}
}

func TestSelectProvider_SixtyDBOnly(t *testing.T) {
	resetProviderEnv(t)
	t.Setenv("SIXTYDB_API_KEY", "sd-key")

	_, name, err := selectProvider()
	if err != nil {
		t.Fatalf("selectProvider error: %v", err)
	}
	if name != providerSixtyDB {
		t.Fatalf("expected %q, got %q", providerSixtyDB, name)
	}
}

func TestSelectProvider_BothPrefersElevenLabs(t *testing.T) {
	resetProviderEnv(t)
	t.Setenv("ELEVENLABS_API_KEY", "el-key")
	t.Setenv("SIXTYDB_API_KEY", "sd-key")

	_, name, err := selectProvider()
	if err != nil {
		t.Fatalf("selectProvider error: %v", err)
	}
	if name != providerElevenLabs {
		t.Fatalf("expected ElevenLabs to win tiebreak, got %q", name)
	}
}

func TestSelectProvider_NeitherErrors(t *testing.T) {
	resetProviderEnv(t)

	_, _, err := selectProvider()
	if err == nil {
		t.Fatal("expected error when no API key is set")
	}
}

func TestEnsureProviderConfigured(t *testing.T) {
	resetProviderEnv(t)
	if err := ensureProviderConfigured(); err == nil {
		t.Fatal("expected error with no keys")
	}
	t.Setenv("SIXTYDB_API_KEY", "sd-key")
	if err := ensureProviderConfigured(); err != nil {
		t.Fatalf("expected 60db key to satisfy configuration, got %v", err)
	}
}
