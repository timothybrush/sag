---
title: Output & formats
description: "Save audio with sag: extension inference, supported ElevenLabs format strings, and tradeoffs."
---

# Output & formats

`-o/--output <path>` writes the generated audio to disk while the stream plays (or instead of playing it). Format selection has two layers:

1. **Extension inference** — `sag` looks at the file extension and picks a sensible ElevenLabs format string.
2. **Explicit override** — `--format <string>` always wins.

## Extension inference

| Path ends with | Inferred `--format` | Notes |
| --- | --- | --- |
| `.mp3` | `mp3_44100_128` | Default; small files, broadly compatible. |
| `.wav` / `.wave` | `pcm_44100` | Lossless PCM at 44.1 kHz. Bigger files. |
| `.ogg` / `.opus` | `opus_48000_64` | Modern, compact, great quality at low bitrate. |

If the extension isn’t recognised, sag falls back to the configured default (`mp3_44100_128`) unless you override `--format`.

```bash
sag -o intro.mp3   "MP3, 128 kbps."
sag -o intro.wav   "Lossless PCM."
sag -o intro.opus  "Opus, 64 kbps."
sag -o weird.bin --format mp3_22050_32 "Cheaper MP3 at 22 kHz."
```

## Common format strings

ElevenLabs publishes the canonical list at <https://elevenlabs.io/docs/api-reference/text-to-speech>. Useful ones for `--format`:

| String | Codec | Sample rate | Bitrate | When to use |
| --- | --- | --- | --- | --- |
| `mp3_44100_128` | MP3 | 44.1 kHz | 128 kbps | Default. Browser-friendly, small. |
| `mp3_44100_192` | MP3 | 44.1 kHz | 192 kbps | Slightly cleaner highs at 1.5× the size. |
| `mp3_22050_32` | MP3 | 22.05 kHz | 32 kbps | Voice-only, tiny files (push notifications). |
| `pcm_44100` | PCM (WAV) | 44.1 kHz | uncompressed | Lossless, post-production master. |
| `pcm_24000` | PCM (WAV) | 24 kHz | uncompressed | Smaller PCM, still beats compressed MP3. |
| `pcm_16000` | PCM (WAV) | 16 kHz | uncompressed | ASR-friendly mono, telephony. |
| `opus_48000_64` | Opus | 48 kHz | 64 kbps | Web/mobile delivery, best quality-per-byte. |
| `ulaw_8000` | μ-law | 8 kHz | 64 kbps | Twilio / classic telephony. |

Some formats require a paid plan; ElevenLabs returns `403` if your tier can’t request them. `sag` surfaces the API error verbatim.

## Streaming vs. format

- **MP3** streams cleanly through both player backends and is the safest default.
- **PCM/WAV** technically streams, but afplay needs the WAV header at the end of the file before it can play; for `--play` you usually want `--no-stream` + `-o file.wav` and listen via `afplay file.wav`.
- **Opus** streams over `afplay` on macOS. The pure-Go `oto` backend doesn’t decode Opus today, so on Linux save to disk and play it with `mpv`, `paplay`, or `vlc`.

## Inspecting what you saved

Quick audio sanity check:

```bash
ffprobe -v quiet -show_entries format=duration,bit_rate,format_name -of csv=p=0 intro.mp3
```

Quick listen:

```bash
ffplay -autoexit -nodisp intro.mp3
afplay intro.mp3   # macOS
```

If duration is short, you probably hit a parent-process timeout — see [Configuration](configuration.md).

## Saving and playing simultaneously

`-o` disables playback by default to avoid surprising agents. Pass `--play` to keep both:

```bash
sag -o intro.mp3 --play "Save and listen at the same time."
```

The two writers share a single read of the response body, so you’re not paying double for bandwidth.

## File-output gotchas

- Sag creates the parent directory (`mkdir -p`) before opening the file. Permissions follow your umask.
- Existing files are truncated and overwritten without prompting. Use a unique name per run if you need history.
- If the request errors mid-stream, the partial file is left in place. Re-run with the same arguments to overwrite.

## Related pages

- [Streaming & playback](streaming.md) — how playback and file writes interact.
- [Speaking text](speak.md) — every `speak` flag.
- [Models](models.md) — engine differences for long-form audio.
