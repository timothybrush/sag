---
title: Install
description: "Install sag via Homebrew, prebuilt release binaries, or `go install`."
---

# Install

`sag` ships as a single static binary. Pick the path that matches your platform.

## Homebrew (macOS, Linux)

```bash
brew install steipete/tap/sag   # auto-taps steipete/tap
sag --version
```

Upgrades:

```bash
brew update && brew upgrade steipete/tap/sag
```

## Prebuilt release binaries

Each tagged release publishes archives for macOS (amd64/arm64), Linux (amd64/arm64), and Windows (amd64). Browse <https://github.com/steipete/sag/releases/latest>, download the matching archive, extract, and put `sag` on your `PATH` (e.g. `/usr/local/bin`).

```bash
curl -L https://github.com/steipete/sag/releases/latest/download/sag_linux_amd64.tar.gz \
  | tar -xz -C /tmp
sudo install -m 0755 /tmp/sag /usr/local/bin/sag
```

## Go toolchain

```bash
go install github.com/steipete/sag/cmd/sag@latest
```

Requires the Go version declared in `go.mod` (1.24+). Source builds bake the Git description into the version string.

## From source

```bash
git clone https://github.com/steipete/sag.git
cd sag
go build ./cmd/sag
./sag --version
```

## Linux build prerequisites

The cross-platform `oto` audio backend needs ALSA development headers when building from source on Debian/Ubuntu:

```bash
sudo apt install build-essential pkg-config libasound2-dev
```

Released Linux binaries already include the audio backend; this step is only required when you compile yourself.

## Verify the install

```bash
sag --version
sag --help
sag prompting   # works without an API key
```

A live API call (any TTS or `sag voices`) needs `ELEVENLABS_API_KEY` set; see [Configuration](configuration.md).

## Updating

- **Homebrew:** `brew upgrade steipete/tap/sag`.
- **Prebuilt archives:** download the new tarball/ZIP and replace the binary.
- **`go install`:** rerun `go install github.com/steipete/sag/cmd/sag@latest`.
- **Source builds:** `git pull && go build ./cmd/sag`.

## Related pages

- [Quickstart](quickstart.md) — first speech in under a minute.
- [Configuration](configuration.md) — API key, env vars, default voice, timeouts.
- [Releasing](RELEASING.md) — the maintainer flow for cutting a new version.
