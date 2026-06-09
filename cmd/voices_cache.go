package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/steipete/sag/internal/tts"
)

const (
	voiceCacheTTL         = 24 * time.Hour
	voiceCacheDirName     = "sag"
	voiceCacheFileName    = "voices.json"
	voiceFetchConcurrency = 4
)

type voiceCache struct {
	Version int                    `json:"version"`
	Voices  map[string]cachedVoice `json:"voices"`
}

type cachedVoice struct {
	Voice     tts.Voice `json:"voice"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newVoiceCache() *voiceCache {
	return &voiceCache{
		Version: 1,
		Voices:  map[string]cachedVoice{},
	}
}

func voiceCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil || home == "" {
			return "", errors.New("no cache directory available")
		}
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, voiceCacheDirName, voiceCacheFileName), nil
}

func loadVoiceCache(path string) (*voiceCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newVoiceCache(), nil
		}
		return nil, err
	}
	cache := newVoiceCache()
	if err := json.Unmarshal(data, cache); err != nil {
		return newVoiceCache(), nil
	}
	if cache.Voices == nil {
		cache.Voices = map[string]cachedVoice{}
	}
	return cache, nil
}

func saveVoiceCache(path string, cache *voiceCache) error {
	if cache == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func hydrateVoices(ctx context.Context, client tts.Provider, voices []tts.Voice, cache *voiceCache, ttl time.Duration) ([]tts.Voice, int) {
	if ttl <= 0 {
		ttl = voiceCacheTTL
	}
	if cache == nil {
		cache = newVoiceCache()
	}

	results := make([]tts.Voice, len(voices))
	now := time.Now()

	var metaCount int
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, voiceFetchConcurrency)

	for i, v := range voices {
		if v.VoiceID == "" {
			results[i] = v
			continue
		}
		if cached, ok := cache.Voices[v.VoiceID]; ok && now.Sub(cached.UpdatedAt) < ttl {
			results[i] = mergeVoice(v, cached.Voice)
			metaCount++
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(index int, voice tts.Voice) {
			defer wg.Done()
			defer func() { <-sem }()

			details, err := client.GetVoice(ctx, voice.VoiceID)
			if err != nil {
				results[index] = voice
				return
			}
			merged := mergeVoice(voice, details)
			results[index] = merged

			mu.Lock()
			cache.Voices[voice.VoiceID] = cachedVoice{
				Voice:     merged,
				UpdatedAt: now,
			}
			metaCount++
			mu.Unlock()
		}(i, v)
	}

	wg.Wait()
	return results, metaCount
}

func mergeVoice(base tts.Voice, details tts.Voice) tts.Voice {
	merged := base
	if details.VoiceID != "" {
		merged.VoiceID = details.VoiceID
	}
	if details.Name != "" {
		merged.Name = details.Name
	}
	if details.Category != "" {
		merged.Category = details.Category
	}
	if details.Description != "" {
		merged.Description = details.Description
	}
	if len(details.Labels) > 0 {
		merged.Labels = details.Labels
	}
	if details.PreviewURL != "" {
		merged.PreviewURL = details.PreviewURL
	}
	return merged
}
