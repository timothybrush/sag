# sag specification

CLI that mirrors macOS `say` but uses ElevenLabs or 60db for synthesis. Defaults to streaming directly to speakers and can also write audio files.

## Runtime & deps
- Go 1.24+
- Playback uses built-in Go audio (go-mp3 + oto) and should work on macOS/Linux/Windows with a default output device.
- Auth via exactly one configured provider: ElevenLabs (`ELEVENLABS_API_KEY`, `SAG_API_KEY`, `--api-key`) or 60db (`SIXTYDB_API_KEY`).

## Commands

### `sag speak [text]`
- Text input: pass as args, `-f/--input-file` (use `-` for stdin), or pipe stdin.
- macOS `say` compatibility:
  - `-v/--voice` accepts voice **name** or ID; `?` lists voices.
  - `-r/--rate` words-per-minute (default 175) maps to ElevenLabs speed.
  - `-o/--output` same meaning; format inferred by extension when possible.
  - Accepts but ignores `--progress`, `--audio-device`, `--network-send`, `--interactive`, `--file-format`, `--data-format`, `--channels`, `--bit-rate`, `--quality`.
- Required: voice (via `-v/--voice` or the active provider default resolution).
- Flags:
  - `--model-id` (default `eleven_v3`; common: `eleven_multilingual_v2`, `eleven_flash_v2_5`, `eleven_turbo_v2_5`)
  - `--format` (default `mp3_44100_128`; `.wav` infers `pcm_44100`)
  - `--stream/--no-stream` (default stream)
  - `--latency-tier` (0-4, default 0)
  - `--play/--no-play` (default play)
  - `--speed` (0.5–2.0, default 1.0; >1.0 speaks faster)
  - `--stability` (0..1; when set)
  - `--similarity` / `--similarity-boost` (0..1; when set)
  - `--style` (0..1; when set)
  - `--speaker-boost` / `--no-speaker-boost`
  - `--seed` (0..4294967295; when set)
  - `--normalize` (`auto|on|off`; when set)
  - `--lang` (2-letter ISO 639-1; when set)
  - `--metrics` print basic stats to stderr
  - `--output <path>` save audio while optionally playing
- Behavior:
  - ElevenLabs streaming path calls `POST /v1/text-to-speech/{voice_id}/stream` with JSON body.
  - 60db streaming path calls `POST /tts-stream` and decodes the documented NDJSON chunk stream.
  - ElevenLabs non-streaming path calls `POST /v1/text-to-speech/{voice_id}` and then plays/saves.
  - 60db non-streaming path calls `POST /tts-synthesize` and decodes `audio_base64`.
  - Errors if neither playback nor output is selected.

Usage examples:
```
sag speak --voice-id VOICE_ID "Hello world"
echo "piped input" | sag speak --voice-id VOICE_ID
sag speak --voice-id VOICE_ID --output out.mp3 --no-play
sag speak --voice-id VOICE_ID --speed 1.15 "Talk a bit faster"
sag speak --voice-id VOICE_ID --stream --latency-tier 3 "Faster start"
sag speak -v "Roger" -r 200 "mac say style flags"
```

### `sag voices`
- Lists voices via the active provider. ElevenLabs uses `GET /v1/voices`; 60db merges `GET /default-voices` and `GET /myvoices`.
- Flags:
  - `--search <query>`: search by name (server-side when available)
  - `--query <text>`: semantic query across name/description/labels (client-side)
  - `--label key=value`: filter by voice label (repeatable)
  - `--limit <n>`: truncate output (default 100)
  - `--try`: play preview audio for the listed voices (requires `--search`, `--query`, `--label`, or `--limit`)

Sample:
```
sag voices --search "english"
```

### `sag prompting`
- Prints a practical prompting guide (model-specific tips, tags, and suggested flags).
- Does not require an API key.

## Config sources
- Exactly one provider key is required.
- ElevenLabs auth: `ELEVENLABS_API_KEY`, `SAG_API_KEY`, `--api-key`, `--api-key-file`.
- 60db auth: `SIXTYDB_API_KEY`, `SIXTYDB_API_KEY_FILE`.
- ElevenLabs default voice env: `ELEVENLABS_VOICE_ID` or `SAG_VOICE_ID`.
- `--base-url` flag for an alternate provider API host.

## Notes & future polish
- Add cross-platform playback backends.
- Persist defaults in a config file (voice/model/format).
- Add tests around flag parsing and error handling.
