---
title: Timeouts & long generations
description: "Why sag has no internal timeout, when to set one, and how to verify long ElevenLabs generations."
---

# Timeouts & long generations

ElevenLabs’ v3 model can take more than a minute on long prompts — sometimes several. Earlier `sag` releases tried to be helpful with hidden 60s/90s deadlines and ended up truncating valid audio. The current behaviour is the simpler one: **no internal timeout by default**.

## How sag picks a timeout

| Source | Behaviour | Default |
| --- | --- | --- |
| `--timeout <duration>` flag | Wraps the request in `context.WithTimeout`. | not set |
| `SAG_TIMEOUT=<duration>` env | Same as the flag, applied when the flag is unset. | not set |
| Parent context | If neither is set, `sag` uses the shell’s context. | inherited |

`<duration>` accepts any value `time.ParseDuration` understands: `30s`, `2m`, `1h30m`. `0` explicitly means “no internal timeout”.

```bash
sag --timeout 5m --no-play -o long.mp3 "$(<chapter.txt)"
SAG_TIMEOUT=10m sag --no-play -o chapter.mp3 "$(<chapter.txt)"
sag --timeout 0 --no-play -o forever.mp3 "$(<book.txt)"
```

## When to set one

- **Agent / scripted runs** where a stuck request would block downstream work. Use a generous outer bound (5–10 minutes) so v3 has room.
- **CI jobs** where you’d rather fail fast than hang the worker.
- **Live UIs** with their own progress indicator, where you want to surface a friendly “try again” after a fixed wait.

## When to leave it off

- One-off interactive runs where you can `Ctrl-C` if it stalls.
- Long-form chapter generation where the worst outcome of a missing timeout is waiting a bit longer.

## Verifying generated audio

After long generations, check the file before publishing. A truncated audio looks fine until someone listens past minute 7.

```bash
ffprobe -v quiet -show_entries format=duration -of csv=p=0 chapter.mp3
```

If the duration is shorter than expected:

1. Check whether your shell, CI runner, or supervisor kicked the process. A parent SIGTERM truncates the partial file in place — sag doesn’t delete it.
2. Re-run with `--metrics` to see chars/bytes/duration on stderr.
3. Increase `--timeout` (or unset it) and rerun.

## Streaming caveat

When streaming (`--stream`, the default), sag writes audio bytes to disk as they arrive. If the request errors halfway, you get a half-formed MP3/WAV/Opus file. Either delete it and retry, or use `--no-stream` so the file appears only after the full response has been received.

## Outer process timeouts

If you also run sag inside a `timeout(1)`, set the outer bound higher than `--timeout`:

```bash
timeout 12m sag --timeout 10m --no-play -o long.mp3 "$(<book.txt)"
```

Pick the inner timeout to bound the request and the outer one to bound the playback + I/O.

## Related pages

- [Configuration](configuration.md) — env vars including `SAG_TIMEOUT`.
- [Streaming & playback](streaming.md) — how cancellation interacts with the player.
- [Models](models.md) — which engines are slow vs. fast.
