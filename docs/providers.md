# Providers

`sag` speaks to two text-to-speech backends behind one consistent CLI. A small
provider abstraction (`internal/tts.Provider`) lets the command layer and the
audio player stay backend-agnostic; each provider translates the shared request
to and from its own wire format.

## Selecting a provider

The provider is auto-detected from whichever API key is present — there is no
`--provider` flag.

| Keys set | Active provider |
|---|---|
| `ELEVENLABS_API_KEY` (or `--api-key`/file) only | ElevenLabs |
| `SIXTYDB_API_KEY` (or `SIXTYDB_API_KEY_FILE`) only | 60db |
| both | ElevenLabs (note printed; unset `ELEVENLABS_API_KEY` to use 60db) |
| neither | error |

Override the host for the active provider with `--base-url`.

## What each provider implements

| Capability | ElevenLabs | 60db |
|---|---|---|
| Auth | `xi-api-key: <key>` | `Authorization: Bearer <key>` |
| Default host | `https://api.elevenlabs.io` | `https://api.60db.ai` |
| List voices | `GET /v1/voices` | `GET /myvoices` |
| Full synthesis | `POST /v1/text-to-speech/{id}` (raw audio) | `POST /tts-synthesize` (base64 in JSON) |
| Streaming | `POST /v1/text-to-speech/{id}/stream` (raw audio) | `POST /tts-stream` (NDJSON, base64 chunks) |

For 60db, `sag` decodes the base64/NDJSON envelope internally, so streaming and
file output behave the same as ElevenLabs. 60db's WebSocket API is not used.

## Flag behavior

Flags are written in ElevenLabs terms and translated per provider so the same
command works on both.

| Flag | ElevenLabs | 60db |
|---|---|---|
| `--speed` / `--rate` | speed multiplier `0.5–2.0` | passthrough (same range) |
| `--stability` | `0..1` | scaled to `0..100` |
| `--similarity` / `--similarity-boost` | `0..1` | scaled to `0..100` |
| `--format` / `-o` extension | `mp3_44100_128`, `pcm_44100`, … | mapped to `mp3` / `wav` / `ogg` / `flac` |
| `--model-id` | model id (e.g. `eleven_v3`) | ignored (model is tied to the voice) |
| `--style` | style exaggeration | ignored |
| `--speaker-boost` / `--no-speaker-boost` | speaker boost | ignored |
| `--seed` | best-effort determinism | ignored |
| `--normalize` | text normalization | ignored |
| `--lang` | language code | ignored |
| `--latency-tier` | streaming latency tier | ignored |

When you set an ignored flag while 60db is active, `sag` prints a one-line note
to stderr rather than failing.

## Notes and limits

- **Playback format:** speaker playback decodes MP3, so when playing through
  speakers `sag` requests MP3 from 60db regardless of `--format`. Use
  `--no-play -o file.<ext>` to save other formats.
- **Voice previews:** `sag voices --try` is not available for 60db (its
  `/myvoices` response has no preview URL); the affected voice is skipped with a
  message.
- **Voice metadata:** 60db's per-voice `model` (`60db Fast` / `60db Quality`) is
  exposed as a `model` label, so `--label model=...` and `--query` can match it.
- **Text limit:** 60db caps synthesis at 5,000 characters per request.
