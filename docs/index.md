---
title: Overview
permalink: /
description: "sag is a Go CLI that turns text into ElevenLabs speech the way macOS `say` does. Stream to speakers, save to files, swap voices, choose models — one binary, terminal-first."
---

## Try it

After [installing](install.md) and exporting `ELEVENLABS_API_KEY`, every TTS call is a one-liner.

```bash
# Speak through the speakers, macOS-say style.
sag "Hello world"

# Pipe stdin like say.
echo "Build green. Ship it." | sag

# Pick a voice by name and slow down a touch.
sag -v Roger -r 165 "Welcome back, Peter."

# Save to disk; format inferred from the extension.
sag -o intro.mp3 "Episode 12 — show notes coming up."
sag -o intro.wav "Lossless PCM, same one-liner."

# Browse voices, then audition with --try.
sag voices --search english --limit 5 --try
```

`sag` reads stdin, accepts `-v/-r/-f/-o` exactly like macOS `say`, and skips the `speak` subcommand when the first argument is plain text. Help, prompting tips, and voice listings need no API key.

## What sag does

- **Drop-in `say` replacement.** Same flag shapes (`-v`, `-r`, `-f`, `-o`), same default of streaming to the speakers. Compatibility no-ops (`--progress`, `--audio-device`, `--quality`, …) keep existing scripts working.
- **Stream-while-you-generate.** Audio plays as bytes arrive over `/v1/text-to-speech/{voice}/stream`. Latency tiers let you trade quality for first-byte time.
- **Voice discovery.** Server-side name search, semantic `--query` over name/description/labels, repeatable `--label key=value` filters, plus `--try` to play preview clips for the matches.
- **Every ElevenLabs model.** Defaults to `eleven_v3` for expressive output; switch to `eleven_multilingual_v2`, `eleven_flash_v2_5`, or `eleven_turbo_v2_5` with `--model-id`. Stability, similarity, style, speaker-boost, seed, normalization, and language are all flag-controlled.
- **Format inference.** `.mp3` → `mp3_44100_128`, `.wav` → `pcm_44100`, `.ogg`/`.opus` → `opus_48000_64`. Override with `--format` when you need something else.
- **Cross-platform playback.** macOS uses `afplay` for AirPlay-friendly routing; Linux and Windows fall back to a `go-mp3` + `oto` decoder. Pick explicitly with `--player auto|afplay|oto`.
- **Honest about limits.** No hidden 60s/90s generation timeouts: long v3 prompts run until they finish. Set `--timeout` or `SAG_TIMEOUT` when you want one.

## Pick your path

- **Trying it.** [Install](install.md) → [Quickstart](quickstart.md). One Homebrew command, one API key, you’re speaking.
- **Making it sound right.** [Prompting](prompting.md) covers v3 audio tags, v2 SSML pauses, and the voice-control sliders.
- **Choosing a model.** [Models](models.md) ranks the four engines by cost, latency, and prompting style.
- **Discovering voices.** [Voices](voices.md) walks through search, semantic queries, label filters, and previewing.
- **Saving audio.** [Output & formats](formats.md) explains extension inference, supported formats, and bitrate tradeoffs.
- **Wiring agents and CI.** [Configuration](configuration.md) covers env vars, key files, timeouts, and the player picker.

## Project

`sag` is MIT-licensed and not affiliated with ElevenLabs or Apple. The [changelog](https://github.com/steipete/sag/blob/main/CHANGELOG.md) tracks releases; the [spec](spec.md) records goals and non-goals. Source: <https://github.com/steipete/sag>.
