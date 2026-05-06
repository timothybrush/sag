---
title: Prompting
description: "Make ElevenLabs sound natural: model-specific tags, SSML pauses, and the voice-control sliders sag exposes."
---

# Prompting

`sag prompting` prints the in-binary cheat sheet (no API key required). This page is the deeper reference.

## Pick the right model first

Prompting style is model-dependent. Match your text to the engine:

- **v3 (alpha) — `eleven_v3`** *(default)*. Inline lowercase audio tags inside square brackets. SSML `<break>` is **not** supported. Pause tags (`[pause]`, `[short pause]`, `[long pause]`) replace it. Short prompts can be unstable; v3 prefers longer scripts (≥250 characters).
- **v2 stable — `eleven_multilingual_v2`**. SSML `<break time="1.5s" />` works for precise pauses up to ~3s. Some English-only models accept SSML `<phoneme>` (not yet exposed by sag).
- **v2.5 Flash — `eleven_flash_v2_5`** and **v2.5 Turbo — `eleven_turbo_v2_5`**. SSML `<break>` supported. Latency-optimized; if normalization sounds off, try `--normalize auto` and respell tricky words.

See [Models](models.md) for the full comparison table.

## Universal “make it sound good” rules

- Write like a script: short sentences, beats on their own line.
- Punctuation is control. Commas + em-dashes slow; ellipses add weight; `!` adds energy.
- Put emphasis in word choice, not stage directions. “quietly” beats “whisper please”.
- For exact pronunciation, respell (e.g. `key-note`); otherwise enable normalization.
- Aim for 250+ characters on v3 — short prompts amplify model variance.

## v3 audio tags

Voice texture:
- `[whispers]`, `[shouts]`
- `[laughs]`, `[starts laughing]`, `[wheezing]`
- `[sighs]`, `[exhales]`, `[clears throat]`
- `[sarcastic]`, `[curious]`, `[excited]`, `[crying]`, `[mischievously]`

Sound effects:
- `[applause]`, `[clapping]`, `[gunshot]`, `[explosion]`
- `[swallows]`, `[gulps]`

Experimental:
- `[strong X accent]` (replace `X`, e.g. `French`)
- `[sings]`

Tag effectiveness depends on the voice + training samples; not every voice reacts well. Don’t pile on — three tags into a paragraph usually beats six.

## v2 / v2.5 SSML breaks

```xml
<break time="0.6s" />
<break time="1.5s" />
```

Up to ~3 seconds. Anything longer is silently clamped. Combine with punctuation rather than relying on SSML alone.

## Sliders, in plain language

| Flag | What it does |
| --- | --- |
| `--stability` | Higher = the voice locks in. Lower = more expressive but jittery. v3 takes 0/0.5/1 (Creative/Natural/Robust); v2/v2.5 takes anything in 0..1. |
| `--similarity` (alias `--similarity-boost`) | Higher = closer to the reference voice; can sound stiff at the top end. |
| `--style` | Higher = more “styled” delivery. Voice/model dependent — some voices ignore it. |
| `--speaker-boost` / `--no-speaker-boost` | Clarity boost; supported on subset of models. |

## Request controls

| Flag | What it does |
| --- | --- |
| `--seed 0..4294967295` | Best-effort repeatability across runs. Not perfect determinism. |
| `--normalize auto\|on\|off` | Numbers/units/URLs normalization. `auto` is a safe default; v2.5 Turbo/Flash sometimes reject `on`. |
| `--lang` | 2-letter ISO 639-1 hint (`en`, `de`, `fr`, …). Influences normalization. |
| `--metrics` | Print chars/bytes/duration to stderr after each call so you can iterate fast. |

## Recipes

Natural narrator (v2):

```bash
sag speak -v Roger \
  --model-id eleven_multilingual_v2 \
  --stability 0.5 --similarity 0.75 --style 0.0 \
  --normalize auto --lang en \
  "We shipped today. It was close — but it worked."
```

Fast and cheap (v2.5 Flash):

```bash
sag speak -v Roger \
  --model-id eleven_flash_v2_5 \
  --stability 0.5 --normalize auto --lang en \
  "Short. Crisp. Low latency."
```

Expressive scene (v3):

```bash
sag speak -v Roger \
  --model-id eleven_v3 \
  --stability 0.5 --normalize off --lang en \
  "[whispers] Don’t move. [short pause] Something’s in the hallway…"
```

Performance with explicit pauses (v2):

```bash
sag speak -v Roger \
  --model-id eleven_multilingual_v2 \
  '<break time="0.5s"/>Five.<break time="0.5s"/>Four.<break time="0.5s"/>Three.<break time="0.5s"/>Two.<break time="0.5s"/>One.'
```

## Iteration loop

When tuning a take, leave `--metrics` on and use `--seed` to keep variance low:

```bash
sag speak -v Roger --metrics --seed 42 \
  --model-id eleven_v3 --stability 0.5 \
  "[curious] Why does this feel so much better today?"
```

If the take is close but slightly off, change one thing per run: drop similarity by 0.05, swap `[curious]` for `[mischievously]`, or shorten a sentence. Stack changes only after you know which knob mattered.

## Related pages

- [Models](models.md) — engine comparison and limits.
- [Speaking text](speak.md) — every flag in one place.
- [Voices](voices.md) — find a voice that responds well to your tags.
