---
title: Speaking text
description: "How sag turns a string into audio: the speak command, default routing, voice selection, and every flag in one place."
---

# Speaking text

`sag speak` is the workhorse subcommand. Most of the time you won’t type it because `sag` auto-routes plain text to `speak` (the same shortcut macOS `say` makes when you skip its subcommand).

## Default routing

These four invocations are equivalent:

```bash
sag "Hello"
sag speak "Hello"
echo "Hello" | sag
echo "Hello" | sag speak
```

Routing rules, applied in `cmd/root.go`:

- If `argv[1]` is a known subcommand (`speak`, `voices`, `prompting`, `help`, `completion`) or a top-level flag, sag runs it directly.
- If `argv[1]` is `--`, sag drops it (npm/pnpm pass-through) and re-runs the dispatch.
- Otherwise, `speak` is injected so flag parsing stays consistent.
- With **no args** but piped stdin, sag also treats it as `speak`.

## Text input precedence

1. `--input-file <path>` (use `-` for stdin) — matches `say -f`.
2. Positional arguments joined with spaces.
3. Stdin, if it isn’t a TTY.

If none of those produce text, `sag` exits with an error rather than calling the API with an empty body.

```bash
sag -f script.txt
sag -f - < script.txt
echo "From a pipe" | sag
sag "Inline string spanning multiple words"
```

## Voice resolution

`-v/--voice` accepts a name **or** an ID. `--voice-id` forces the value to be treated as an ID even if it looks like a name. Resolution order:

1. Flag value (`-v` or `--voice-id`).
2. `ELEVENLABS_VOICE_ID` env.
3. `SAG_VOICE_ID` env.
4. First voice from `/v1/voices` (warning printed to stderr).

Pass `?` to print the voice table and exit:

```bash
sag -v ?
```

Name lookup is case-insensitive, with exact matches preferred over substring matches. If multiple voices share a name, sag picks the first hit deterministically and logs the choice.

## Streaming vs. non-streaming

Streaming is the default. Audio plays as bytes arrive over `POST /v1/text-to-speech/{voice}/stream`. With `--no-stream`, `sag` falls back to `POST /v1/text-to-speech/{voice}`, which returns the entire payload before playback starts (useful for tests or when a downstream tool needs the full file before any byte hits the speakers).

Those routes and streaming controls apply to ElevenLabs. With 60db, sag uses `/tts-synthesize`, buffers and validates the live NDJSON response, wraps its PCM as WAV, and then plays or saves it. The default stream mode is disabled automatically; explicit `--stream` is rejected.

```bash
sag --stream "Default."
sag --no-stream -o file.mp3 "Wait, then save."
sag --latency-tier 3 "First-byte time matters more than fidelity."
```

See [Streaming & playback](streaming.md) for the trade-offs.

## Speed and rate

| Flag | Range | Default | Notes |
| --- | --- | --- | --- |
| `--speed` | 0.5–2.0 | 1.0 | Direct ElevenLabs multiplier. |
| `-r/--rate` | words-per-minute | 175 (`say` default) | Mapped to `--speed` as `wpm/175`. |

`-r` overrides `--speed` when both are set. The mapped speed must still fall within 0.5–2.0; rates that produce values outside the range error out clearly.

## Voice control sliders

| Flag | Range | Behaviour |
| --- | --- | --- |
| `--stability` | 0/0.5/1 on v3; 0..1 on v2/v2.5 | Higher = more consistent, less expressive. v3 enforces the three presets (Creative/Natural/Robust). |
| `--similarity` (alias `--similarity-boost`) | 0..1 | Higher = closer to the reference voice; can sound stiff at the top end. |
| `--style` | 0..1 | More “styled” delivery; voice/model dependent. |
| `--speaker-boost` / `--no-speaker-boost` | toggle | Clarity boost; supported only on some models. |

Sliders are only sent when explicitly set (`flag.Changed`). That keeps server-side defaults intact when you don’t care.

## Request controls

| Flag | Behaviour |
| --- | --- |
| `--seed 0..4294967295` | Best-effort reproducibility across runs. |
| `--normalize auto\|on\|off` | Numbers/units/URLs normalization. v2.5 Turbo/Flash sometimes reject `on`; use `auto`. |
| `--lang` | 2-letter ISO 639-1 hint (`en`, `de`, `fr`, …). |
| `--metrics` | Print `chars=… bytes=… model=… voice=… stream=… latencyTier=… dur=…` to stderr after each call. |

## Output

`-o/--output <path>` writes generated audio to disk. The format is inferred from the extension:

| Extension | Format string |
| --- | --- |
| `.mp3` | `mp3_44100_128` |
| `.wav` / `.wave` | `pcm_44100` |
| `.ogg` / `.opus` | `opus_48000_64` |

Override with `--format mp3_22050_32` (or any string ElevenLabs accepts). When `-o` is set, speaker playback is disabled by default — pass `--play` to keep both, or `--no-play` to be explicit.

60db output is 48 kHz mono WAV only, so use `.wav` or `--format wav`.

## Putting it together

```bash
# Voice-acted v3 with a save to disk and metrics on stderr.
sag speak \
  -v Roger \
  --model-id eleven_v3 \
  --stability 0.5 --style 0.4 \
  --metrics \
  -o scene.mp3 \
  "[whispers] Don’t move. [short pause] Something’s in the hallway."
```

## Related pages

- [Voices](voices.md) — finding and previewing voices.
- [Prompting](prompting.md) — model-specific tips and tags.
- [Models](models.md) — pick the right engine.
- [Streaming & playback](streaming.md) — latency tiers and player backends.
- [Output & formats](formats.md) — codecs and format strings.
