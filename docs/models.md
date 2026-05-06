---
title: Models
description: "Pick the right ElevenLabs engine: v3, v2 multilingual, v2.5 Flash, or v2.5 Turbo — with prompting style and limits."
---

# Models

`sag` passes any value you give it to `--model-id` straight through to ElevenLabs. There is no allow-list, so new model IDs work the moment ElevenLabs ships them. The defaults and notes below cover the ones you’ll actually pick.

## At a glance

| Engine | `--model-id` | Prompting style | Best for | Input cap |
| --- | --- | --- | --- | --- |
| v3 (alpha) | `eleven_v3` *(default)* | Audio tags `[whispers]`, `[short pause]` (no SSML breaks) | Most expressive / “acting” | ~5,000 chars |
| v2 (stable) | `eleven_multilingual_v2` | SSML `<break>` supported | Reliable baseline, simple prompts | ~10,000 chars |
| v2.5 Flash | `eleven_flash_v2_5` | SSML `<break>` supported | Ultra-low latency (~75ms), 50% cheaper per char | ~40,000 chars |
| v2.5 Turbo | `eleven_turbo_v2_5` | SSML `<break>` supported | Low latency (~250–300ms), 50% cheaper per char | ~40,000 chars |

ElevenLabs’ docs are the source of truth for limits and pricing — these are practical defaults that match what `sag prompting` shows.

## Picking a model

- **Default to v3** when you want voice acting. Audio tags and pause tags give you control SSML can’t express.
- **Drop to v2** when v3 jitters on a short prompt or you need precise SSML pauses. v2 is the calmest baseline.
- **Use v2.5 Flash** for low-latency live agents, push-to-talk, or anywhere TTFB matters more than fidelity.
- **Use v2.5 Turbo** when you want most of Flash’s latency wins but a touch more headroom on quality.

When you switch models, revisit the sliders. v3 only accepts `0/0.5/1` for `--stability`; v2/v2.5 take any 0..1. v3 ignores `<break>`, v2/v2.5 ignore audio tags.

## Switching at the CLI

```bash
sag speak -v Roger --model-id eleven_v3 \
  "[whispers] Stay close."

sag speak -v Roger --model-id eleven_multilingual_v2 \
  '<break time="0.4s"/>Stay close.'

sag speak -v Roger --model-id eleven_flash_v2_5 \
  "Status update: build green, queue empty."
```

## Long-form text

If your input exceeds the engine’s cap:

1. Chunk along sentence boundaries (don’t cut mid-clause).
2. Generate each chunk to a numbered file with `-o chunk_NN.mp3`.
3. Stitch with `ffmpeg -f concat`. Use the same `--seed` for every chunk to reduce voice drift.

```bash
ffmpeg -f concat -safe 0 -i list.txt -c copy chapter.mp3
```

For v3 specifically, prefer fewer, longer chunks: short chunks make v3 less stable.

## Normalization quirks

`--normalize on` may error on v2.5 Turbo/Flash. Stick with `auto` (or omit the flag) unless you have a strong reason. See [Prompting](prompting.md) for the full slider tour.

## Verifying generated audio

Streamed and chunked output can be hard to eyeball. After long generations:

```bash
ffprobe -v quiet -show_entries format=duration -of csv=p=0 chapter.mp3
```

If the duration is shorter than expected, you probably hit a parent-process timeout — set `--timeout` or `SAG_TIMEOUT` (see [Configuration](configuration.md)).

## Related pages

- [Prompting](prompting.md) — model-specific tags and slider recipes.
- [Speaking text](speak.md) — every `speak` flag.
- [Output & formats](formats.md) — codecs, bitrates, file extensions.
