---
title: macOS say compatibility
description: "How sag mirrors the macOS say command — same flag shapes, same default of streaming to speakers."
---

# macOS `say` compatibility

`sag` reads like a drop-in for Apple’s `say`. Existing scripts and aliases mostly work without changes; the differences are intentional and small.

## Shared shape

| `say` | `sag` | Notes |
| --- | --- | --- |
| `say "Hello"` | `sag "Hello"` | Plain text auto-routes to `speak`. |
| `echo Hi \| say` | `echo Hi \| sag` | Stdin works without a subcommand. |
| `say -v Daniel "Hi"` | `sag -v Roger "Hi"` | `-v` accepts ElevenLabs voice names or IDs. |
| `say -v ?` | `sag -v ?` | List voices and exit. |
| `say -r 200 "Faster"` | `sag -r 200 "Faster"` | Words-per-minute, default 175. |
| `say -f script.txt` | `sag -f script.txt` | File input. `-` reads stdin. |
| `say -o out.aiff` | `sag -o out.mp3` | sag writes MP3/WAV/OGG/Opus depending on extension. |

## Accepted no-op flags

These flags exist on `say` but don’t map cleanly to a server-side TTS. `sag` accepts them and silently ignores them so your scripts keep running:

`--progress`, `--audio-device`, `--network-send`, `--interactive`, `--file-format`, `--data-format`, `--channels`, `--bit-rate`, `--quality`.

## Differences worth knowing

- **`-o` defaults to file-only.** `say -o file.aiff` plays through speakers AND writes the file. `sag -o file.mp3` writes the file and skips playback unless you pass `--play`.
- **No AIFF.** `say` defaults to AIFF. `sag` defaults to MP3 (44.1 kHz / 128 kbps) and infers WAV/Opus from `.wav`/`.opus`.
- **Voice catalog is online.** `say -v ?` shows local SpeechSynthesisManager voices. `sag -v ?` calls ElevenLabs and prints what your API key can use.
- **Rate semantics.** `say -r` is words-per-minute relative to a built-in voice baseline; `sag -r` maps to ElevenLabs’ `--speed` multiplier (`speed = wpm / 175`). The mapped value must stay within ElevenLabs’ 0.5–2.0 range or `sag` errors out clearly.
- **No `-i` / interactive mode.** Accepted as a no-op for compatibility.
- **No `--audio-device` routing.** Output goes through the platform default (or your AirPlay/Bluetooth selection on macOS, via `afplay`).

## Drop-in alias

If you want `say` to mean `sag` in a shell:

```bash
# zsh / bash
alias say='sag'
```

Most one-liners (`say -v Daniel "Hello"`, `say -f script.txt`, `say -r 200 "Faster"`) keep working. Replace voice names that no longer exist in ElevenLabs, and adjust `-o` paths if you relied on AIFF.

## Why the parity matters

The `say`-style ergonomics are the whole point of `sag`: muscle memory transfers, scripts keep working, and you can swap the underlying TTS engine without rewriting your tooling.

## Related pages

- [Speaking text](speak.md) — every `speak` flag in one place.
- [Voices](voices.md) — find an ElevenLabs voice that fits the character you used to ask `say` for.
- [Streaming & playback](streaming.md) — `--play`, `--no-play`, and the player backends.
