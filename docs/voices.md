---
title: Voices
description: "Discover ElevenLabs voices with name search, semantic queries, label filters, and audio previews."
---

# Voices

`sag voices` lists the voices available to your API key. The same engine drives `-v` resolution: anything you can find with `voices` you can speak with by name.

## Quick listing

```bash
sag voices                  # all voices, capped at 100 rows
sag voices --limit 10       # first 10
sag voices --limit 0        # no cap
```

The output is plain TSV with a header (`VOICE ID`, `NAME`, `CATEGORY`), aligned with `tabwriter`. Pipe it into `awk`, `column`, or your favourite picker.

## Server-side name search

```bash
sag voices --search english --limit 20
sag voices --search "narrator"
```

`--search` first tries the ElevenLabs `/voices` endpoint with the search term. If the server doesn’t honour the query (older API versions, custom proxies), `sag` falls back to a client-side substring match against the full voice list — same UX, slightly more bandwidth.

## Semantic query

`--query` ranks voices by similarity over **name + description + labels**, using the cached metadata. It’s great for vibes-based browsing.

```bash
sag voices --query "crazy scientist" --limit 5
sag voices --query "warm grandmother bedtime story" --limit 5
sag voices --query "calm british narrator" --limit 5 --try
```

If the metadata cache hasn’t been hydrated yet, `sag` fetches detail pages on demand and prints `warning: voice metadata unavailable; matching on names only` when the API doesn’t return labels.

## Label filters

ElevenLabs voices carry structured labels (accent, age, use_case, gender, …). Filter by repeating `--label`:

```bash
sag voices --label accent=british --limit 20
sag voices --label use_case=character --label gender=male --limit 10
```

Multiple `--label` flags are AND-combined. Values are compared case-insensitively.

## Audio previews

`--try` plays the preview clip for each listed voice. To avoid blasting every voice in your account, `--try` requires at least one of `--search`, `--query`, `--label`, or an explicit `--limit`:

```bash
sag voices --search english --limit 5 --try
sag voices --query "calm storyteller" --limit 3 --try
sag voices --label accent=irish --try
```

Previews are streamed via the same player backend as `speak` (`--player auto|afplay|oto`, see [Streaming & playback](streaming.md)). Failed previews don’t abort the loop — sag logs and continues.

## Combining with `speak`

Once you find a voice you like:

```bash
sag voices --query "calm narrator" --limit 1
# VOICE ID                  NAME    CATEGORY
# 21m00Tcm4TlvDq8ikWAM      Roger   premade

sag -v Roger "Welcome to chapter one."
sag --voice-id 21m00Tcm4TlvDq8ikWAM "Same voice, by ID."
```

For repeated runs in the same shell:

```bash
export SAG_VOICE_ID=21m00Tcm4TlvDq8ikWAM
```

## Cache lifetime

`sag voices --query` and `--label` cache the hydrated voice list for 24 hours. Location follows the OS config dir (`$XDG_CONFIG_HOME/sag/voices.json` on Linux, `~/Library/Application Support/sag/voices.json` on macOS, `%AppData%\sag\voices.json` on Windows). Delete the file to force a fresh fetch.

## Tips

- Newly cloned voices may take a minute to appear; rerun `sag voices` after the dashboard confirms training.
- Some preview URLs 404 — usually a voice was just unpublished. Use `--query` to get a substitute.
- `voices --query "..."` is a better starting point than `--search` when you’re browsing rather than looking for an exact name.

## Related pages

- [Speaking text](speak.md) — pass the chosen voice to `speak`.
- [Configuration](configuration.md) — `SAG_VOICE_ID` and friends.
- [Streaming & playback](streaming.md) — preview playback uses the same backend stack.
