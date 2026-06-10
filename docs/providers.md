# Providers

`sag` supports two HTTP TTS backends. The CLI auto-selects the provider from your configured credentials; there is no `--provider` flag.

## Selecting a provider

| Keys set | Active provider |
| --- | --- |
| `ELEVENLABS_API_KEY` / `SAG_API_KEY` or `--api-key` / `--api-key-file` only | ElevenLabs |
| `SIXTYDB_API_KEY` or `SIXTYDB_API_KEY_FILE` only | 60db |
| both | error |
| neither | error |

Use `--base-url` to override the active provider host.

## 60db routes sag uses

The 60db integration is deliberately limited to the documented REST contract:

| Capability | Route | Notes |
| --- | --- | --- |
| Voice listing | `GET /default-voices` | Workspace-default voices; queried first. |
| Voice listing | `GET /myvoices` | User-created voices; appended after defaults and deduped by `voice_id`. |
| Streaming speak | `POST /tts-stream` | NDJSON stream of base64 chunks; no `output_format` field in the request. |
| Full speak | `POST /tts-synthesize` | JSON envelope with `audio_base64`, `encoding`, and `output_format`. |

For 60db, `sag` validates the JSON `success` envelope even on HTTP 200 responses, decodes base64/NDJSON audio internally, rejects malformed or empty audio, and enforces per-chunk, per-frame, and total-audio limits.

## Flag behavior

### Supported on both providers

- `-v, --voice`
- `-r, --rate`
- `--speed`
- `--stability`
- `--similarity` / `--similarity-boost`
- `--format`
- `--stream` / `--no-stream`
- `--play` / `--no-play`
- `-o, --output`
- `--timeout`
- `--metrics`

### ElevenLabs-only

These flags are passed through to ElevenLabs and rejected on 60db:

- `--model-id`
- `--style`
- `--speaker-boost`
- `--no-speaker-boost`
- `--seed`
- `--normalize`
- `--lang`
- `--latency-tier`

### Parameter translation

- `--stability` and `--similarity` stay in the CLI's `0..1` range and are scaled to 60db's documented `0..100` request values.
- `--format` is canonicalized for 60db full synthesis: `mp3_*` → `mp3`, `pcm_*` / `wav` → `wav`, `opus_*` / `ogg` → `ogg`, `flac` → `flac`.

## Streaming, files, and playback

- ElevenLabs can stream in the requested output format, so `--stream` and `--format` work together.
- 60db streaming is used only for the default MP3 path because `/tts-stream` does not document `output_format`.
- On 60db, if the effective output format is non-MP3 and streaming was only enabled by the default, `sag` automatically falls back to `POST /tts-synthesize`.
- On 60db, `--stream` plus a non-MP3 format is an error when you explicitly force `--stream`.
- On 60db, `--play` requires MP3 output. Use `--no-play -o out.wav` (or `ogg` / `flac`) for other formats.

## Voice metadata notes

- `sag voices --try` uses `GET /voices/:id` on 60db to fetch `sample_url` when the list responses do not include preview URLs.
- 60db voice `model` and `categories` values are exposed as CLI labels so `--query` and `--label model=...` still work.
