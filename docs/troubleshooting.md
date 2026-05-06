---
title: Troubleshooting
description: "Common sag failure modes and how to fix them: missing keys, voice not found, partial audio, playback silence."
---

# Troubleshooting

This page collects the fixes you’ll actually need. If something here is wrong or missing, open an issue at <https://github.com/steipete/sag/issues>.

## `ELEVENLABS_API_KEY is not set`

Sag couldn’t find a key via flag, env, or file.

- Set `ELEVENLABS_API_KEY` (or `SAG_API_KEY`) in your shell.
- Or use `--api-key-file ~/.config/elevenlabs.key`.
- Confirm with `printenv ELEVENLABS_API_KEY | head -c 5`.
- See [Configuration](configuration.md) for the full precedence list.

## `voice "X" not found; try 'sag voices' or -v '?'`

The literal voice name isn’t in your account. Common causes:

- Typo. `sag voices --search X` finds it.
- Voice belongs to a different ElevenLabs workspace (paid voices aren’t shared across accounts).
- API key lacks the `voices` scope.

If the voice exists in the dashboard but not in `sag voices`, regenerate the API key with the correct scopes. ElevenLabs caches the old scope set on the key for a few seconds.

## Partial / truncated audio file

The most common cause is a parent timeout. Sag streams audio to disk; if the request is cancelled, the partial file remains.

1. Re-run with `--metrics` to see chars/bytes/duration.
2. Increase `--timeout` (or unset `SAG_TIMEOUT`).
3. Verify after generation:

   ```bash
   ffprobe -v quiet -show_entries format=duration -of csv=p=0 long.mp3
   ```

If the file size grows over time, you’re fine; if `ffprobe` reports a much shorter duration than expected, regenerate. See [Timeouts](timeouts.md).

## Audio plays at the wrong speed

`-r/--rate` maps to ElevenLabs’ speed multiplier as `wpm / 175`. The mapped value must stay within 0.5–2.0. If you see:

```
rate 90 wpm maps to speed 0.51, which is outside the allowed 0.5–2.0 range
```

or similar, lower or raise the rate to match. For finer control, use `--speed` directly (e.g. `--speed 0.92`).

## Speaker is silent (macOS)

If `afplay` is selected and nothing comes out:

1. `afplay /System/Library/Sounds/Funk.aiff` — does the system tone play?
2. If yes, retry with `--player oto` to bypass `afplay`.
3. If `afplay` works but only on built-in speakers, your AirPlay/Bluetooth target may not announce a route fast enough; toggle output device in **System Settings → Sound** and retry.

## Speaker is silent (Linux)

The `oto` backend depends on a working ALSA/PulseAudio stack.

- Test with `paplay /usr/share/sounds/alsa/Front_Center.wav`.
- For containers, expose `--device /dev/snd` and mount the right audio sockets, or run with `--no-play -o /tmp/out.mp3` and play later.

## `mp3_*` format errors on `pcm_*` paths

Some `--format` strings require a paid plan. If you get `403 forbidden`, downgrade to a tier your plan supports (e.g. drop from `mp3_44100_192` to `mp3_44100_128`). See [Output & formats](formats.md).

## `eleven_v3` rejects `--stability 0.7`

v3 only accepts the three presets: `0.0` (Creative), `0.5` (Natural), `1.0` (Robust). Pick one or switch to a v2 model where `0..1` is allowed.

## `--normalize on` errors on v2.5

ElevenLabs sometimes rejects `on` for v2.5 Turbo/Flash. Use `auto` (the safest default) or `off` and respell tricky words yourself.

## `--latency-tier 4` doesn’t feel any faster

Latency tiers reduce decoder overhead and quality knobs. On v2.5 Flash you’ll rarely notice tiers above 1; on v3 the wins flatten out around 2–3. Tier 4 is a cosmetic “lowest possible” setting.

## `unknown player "xyz"`

`--player` only accepts `auto`, `afplay`, or `oto`. The `SAG_PLAYER` env var has the same restriction. See [Streaming & playback](streaming.md).

## Sag aliases conflict with system `say`

If `alias say='sag'` confuses scripts that depend on Apple’s `say`, scope the alias to your interactive shell only:

```bash
# ~/.zshrc
[[ -o interactive ]] && alias say='sag'
```

Or invoke macOS `say` explicitly with `/usr/bin/say` when you need the original.

## Still stuck

- `sag --version` — confirm you’re on the latest release.
- `sag voices --limit 1` — sanity-check API connectivity.
- `sag --metrics` on a known-good prompt — capture the diagnostic line.
- File an issue at <https://github.com/steipete/sag/issues> with the metrics line and the exact command.

## Related pages

- [Configuration](configuration.md)
- [Streaming & playback](streaming.md)
- [Timeouts](timeouts.md)
- [Models](models.md)
