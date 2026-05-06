---
title: Using sag from agents
description: "Recommended patterns for shelling out to sag from coding agents, scripts, and CI."
---

# Using sag from agents

`sag` is built to be called from non-interactive contexts — coding agents, scripts, CI jobs, live audio backends. This page collects the patterns that make automation predictable.

## Stable invocation

```bash
sag --no-play -o "$artifact" "$prompt"
```

- `--no-play` keeps the agent silent. Speaker output rarely makes sense in CI.
- `-o` writes the audio to a known path. The format is inferred from the extension (see [Output & formats](formats.md)).
- The text comes via positional argument, `-f file`, or piped stdin. Pick one and stick with it.

Always check the exit code (`$?`). Non-zero means the API or the file system errored. The error message goes to stderr.

## Always use a key file or env

Don’t put your API key in `argv`. Process listings leak across containers and CI logs.

```bash
export ELEVENLABS_API_KEY=...
# or
export ELEVENLABS_API_KEY_FILE=/run/secrets/elevenlabs.key
```

Either form is fine. The file form is friendlier for Docker secrets and Kubernetes mounts.

## Pin a voice

Resolve the voice once and pass the ID, not the name:

```bash
voice_id=$(sag voices --search "$voice_name" --limit 1 | awk 'NR==2 {print $1}')
sag --voice-id "$voice_id" --no-play -o "$artifact" "$prompt"
```

Names can be ambiguous; IDs are stable and don’t require a list call on every invocation. Cache the ID (env var, config file, agent state) and refresh only when the voice changes.

## Time it

Set `--timeout` (or `SAG_TIMEOUT`) in agent contexts. Without one, a slow ElevenLabs response will block the whole agent step indefinitely.

```bash
SAG_TIMEOUT=5m sag --no-play -o "$artifact" "$prompt"
```

Pick the timeout based on the model:

- **v3** (`eleven_v3`): 3–5 minutes for paragraph-length prompts.
- **v2** (`eleven_multilingual_v2`): 1–2 minutes is usually plenty.
- **v2.5 Flash/Turbo**: 30–60 seconds is often enough.

See [Timeouts](timeouts.md).

## Validate the artifact

Generated audio can be partial. After long runs, verify the duration:

```bash
ffprobe -v quiet -show_entries format=duration -of csv=p=0 "$artifact"
```

Fail the agent step if the duration is below the minimum you expect.

## Be loud about metrics

`--metrics` prints a single line to stderr after each call. Capture it; agents love structured logs:

```text
metrics: chars=812 bytes=325120 model=eleven_v3 voice=21m00Tcm4TlvDq8ikWAM stream=true latencyTier=0 dur=18.4s
```

You can `grep -F 'metrics:'` the stderr stream and dump the line into your run’s metadata.

## Container hints

A minimal Docker invocation:

```bash
docker run --rm \
  -e ELEVENLABS_API_KEY \
  -v "$PWD/out:/out" \
  ghcr.io/your-org/sag-runner:latest \
  sag --no-play -o /out/episode.mp3 "$prompt"
```

The `oto` backend is the default outside macOS, but containers usually have no audio device. `--no-play` (or `SAG_PLAYER` set to anything that doesn’t play) keeps sag from attempting playback.

## Idempotency

`sag` overwrites `-o` files without prompting. For agents that retry on partial output, this is the right behaviour — re-run with the same arguments and the file is replaced. For audit logs, write to versioned paths (`out/run-<timestamp>.mp3`) and let the agent clean up.

## Related pages

- [Configuration](configuration.md) — env vars and key files.
- [Timeouts](timeouts.md) — pick the right `--timeout`.
- [Output & formats](formats.md) — codec + extension matrix.
