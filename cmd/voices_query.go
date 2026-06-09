package cmd

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/steipete/sag/internal/tts"
)

type labelFilter struct {
	key   string
	value string
}

func parseLabelFilters(filters []string) ([]labelFilter, error) {
	if len(filters) == 0 {
		return nil, nil
	}
	parsed := make([]labelFilter, 0, len(filters))
	for _, raw := range filters {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("label filter %q must be key=value", raw)
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.ToLower(strings.TrimSpace(parts[1]))
		if key == "" || value == "" {
			return nil, fmt.Errorf("label filter %q must be key=value", raw)
		}
		parsed = append(parsed, labelFilter{key: key, value: value})
	}
	return parsed, nil
}

func filterVoicesByLabels(voices []tts.Voice, filters []labelFilter) []tts.Voice {
	if len(filters) == 0 {
		return voices
	}
	filtered := make([]tts.Voice, 0, len(voices))
	for _, v := range voices {
		if matchesAllLabels(v, filters) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func matchesAllLabels(voice tts.Voice, filters []labelFilter) bool {
	if len(filters) == 0 {
		return true
	}
	if len(voice.Labels) == 0 {
		return false
	}
	for _, filter := range filters {
		val, ok := labelValue(voice.Labels, filter.key)
		if !ok {
			return false
		}
		if strings.ToLower(strings.TrimSpace(val)) != filter.value {
			return false
		}
	}
	return true
}

func labelValue(labels map[string]string, key string) (string, bool) {
	if labels == nil {
		return "", false
	}
	if val, ok := labels[key]; ok {
		return val, true
	}
	for k, v := range labels {
		if strings.ToLower(k) == key {
			return v, true
		}
	}
	return "", false
}

func rankVoicesByQuery(voices []tts.Voice, query string) []tts.Voice {
	query = strings.TrimSpace(query)
	if query == "" {
		return voices
	}
	tokens := tokenizeQuery(query)
	scored := make([]scoredVoice, 0, len(voices))
	for _, v := range voices {
		score := scoreVoice(v, query, tokens)
		if score > 0 {
			scored = append(scored, scoredVoice{voice: v, score: score})
		}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return strings.ToLower(scored[i].voice.Name) < strings.ToLower(scored[j].voice.Name)
		}
		return scored[i].score > scored[j].score
	})
	ranked := make([]tts.Voice, 0, len(scored))
	for _, s := range scored {
		ranked = append(ranked, s.voice)
	}
	return ranked
}

type scoredVoice struct {
	voice tts.Voice
	score int
}

func scoreVoice(voice tts.Voice, query string, tokens []string) int {
	name := strings.ToLower(voice.Name)
	desc := strings.ToLower(voice.Description)
	labels := strings.ToLower(flattenLabels(voice.Labels))
	category := strings.ToLower(voice.Category)
	queryLower := strings.ToLower(query)

	score := 0
	if queryLower != "" {
		if strings.Contains(name, queryLower) {
			score += 10
		} else if strings.Contains(desc, queryLower) {
			score += 7
		} else if strings.Contains(labels, queryLower) {
			score += 5
		}
	}

	for _, token := range tokens {
		if strings.Contains(name, token) {
			score += 6
		}
		if desc != "" && strings.Contains(desc, token) {
			score += 4
		}
		if labels != "" && strings.Contains(labels, token) {
			score += 3
		}
		if category != "" && strings.Contains(category, token) {
			score += 1
		}
	}
	return score
}

func tokenizeQuery(query string) []string {
	var b strings.Builder
	b.Grow(len(query))
	for _, r := range strings.ToLower(query) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	raw := strings.Fields(b.String())
	tokens := make([]string, 0, len(raw))
	for _, token := range raw {
		if len(token) < 2 {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func flattenLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	var b strings.Builder
	for k, v := range labels {
		if k == "" && v == "" {
			continue
		}
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(v)
		b.WriteByte(' ')
	}
	return b.String()
}
