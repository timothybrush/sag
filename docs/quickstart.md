---
title: Quickstart
description: "From `brew install` to your first streamed sentence in under a minute."
---

# Quickstart

Sixty seconds from a clean machine to a streaming ElevenLabs sentence. For the longer tour, jump to [Prompting](prompting.md), [Voices](voices.md), and [Models](models.md) afterwards.

## 1. Install

```bash
brew install steipete/tap/sag
sag --version
```

Other install paths (`go install`, prebuilt archives, source build) are on [Install](install.md).

## 2. Get an ElevenLabs API key

1. Sign in at <https://elevenlabs.io>.
2. Open **Profile → API Keys** and create a key with the scopes you need (Text-to-Speech, Voices).
3. Copy the key once — it’s only shown at creation time.

## 3. Export the key

```bash
export ELEVENLABS_API_KEY="sk-..."
```

For agents and CI you can keep the key in a file instead of an env var:

```bash
echo "$ELEVENLABS_API_KEY" > ~/.config/elevenlabs.key
chmod 600 ~/.config/elevenlabs.key
sag --api-key-file ~/.config/elevenlabs.key voices --limit 3
# or set ELEVENLABS_API_KEY_FILE / SAG_API_KEY_FILE
```

See [Configuration](configuration.md) for every option.

## 4. Speak something

```bash
sag "Hello from sag."
```

`sag` defaults to `speak` when the first argument is plain text, so the subcommand is optional. Equivalent forms:

```bash
sag speak "Hello from sag."
echo "Piped input works too." | sag
sag -f notes.txt              # read from a file (matches `say -f`)
sag -f -                      # read from stdin
```

## 5. Pick a voice

List voices and audition the first match:

```bash
sag voices --search english --limit 5 --try
```

Then call `speak` with the voice name or ID:

```bash
sag -v Roger "I’ll be your narrator today."
sag --voice-id 21m00Tcm4TlvDq8ikWAM "Use the raw ID directly."
```

Persist a default for the session:

```bash
export SAG_VOICE_ID=21m00Tcm4TlvDq8ikWAM
# or ELEVENLABS_VOICE_ID — both work
```

## 6. Save audio to disk

```bash
sag -o intro.mp3 "Episode 12 — show notes."   # 44.1 kHz / 128 kbps MP3
sag -o intro.wav "Lossless PCM."              # WAV (PCM 44.1 kHz)
sag -o intro.opus "Smaller files."            # Opus 48 kHz / 64 kbps
```

`-o` disables speaker playback unless you also pass `--play`. Format is inferred from the extension; override explicitly with `--format` (see [Output & formats](formats.md)).

## 7. Speed it up

`-r/--rate` matches macOS `say` (words-per-minute, default 175). `--speed` is the underlying ElevenLabs multiplier.

```bash
sag -v Roger -r 220 "A bit faster."
sag -v Roger --speed 0.92 "And a bit slower."
```

## 8. Make it sound better

```bash
sag prompting   # the in-binary cheat sheet, no API key needed
```

The full guide lives at [Prompting](prompting.md). Highlights:

- **v3 (default):** inline tags like `[whispers]`, `[short pause]`, `[laughs]`.
- **v2 / v2.5:** SSML breaks like `<break time="1.2s" />`.
- Use the sliders: `--stability`, `--similarity`, `--style`, `--speaker-boost`.

## Where next

- [Voices](voices.md) — semantic `--query`, label filters, preview playback.
- [Models](models.md) — pick between v3, v2, v2.5 Flash, v2.5 Turbo.
- [Streaming & playback](streaming.md) — latency tiers, player backends, `--no-stream`.
- [Output & formats](formats.md) — supported codecs and extension inference.
- [Configuration](configuration.md) — env vars, key files, timeouts, base URL.
