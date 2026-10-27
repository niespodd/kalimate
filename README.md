<div align="center">

# Kalimate

### A fast, focused kalimba tuner for macOS

Pluck a tine — Kalimate reads the pitch from the *sustain* of the note, shows you
the nearest key, how far off you are, and exactly which way to nudge the tine.

![macOS](https://img.shields.io/badge/macOS-000000?logo=apple&logoColor=white)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Fyne](https://img.shields.io/badge/UI-Fyne-00AC9F)
![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)

<img src="preview.png" alt="Kalimate tuning the C5 tine" width="420">

</div>

---

## Why Kalimate

Chromatic tuners fight a kalimba: the metal tines have a sharp, noisy attack and a
long, slightly inharmonic decay, so a plain per-frame reading jitters and jumps
octaves. Kalimate is built around the way a kalimba actually sounds.

- 🎯 **Reads the pluck, not the noise** — each note is reduced to one steady
  frequency from its sustain, so the number holds still while you adjust.
- 🫳 **Tells you which way to move the tine** — *push in* to raise the pitch,
  *pull out* to lower it. No mental math from a ± cents needle.
- 🗺️ **Shows you which key is which** — a 17-tine kalimba map lights up the key
  you're hearing and turns it green once it's confirmed in tune.
- ✅ **Confirms and moves on** — hold within ±5 cents for a moment and the key
  locks in; hit **Next** (or the spacebar) to walk the whole instrument.
- ⚡️ **Instant and offline** — a single native binary, no accounts, no network.
- 🌍 **Localized** — UI strings run through Fyne's i18n and follow your system
  language.

## Install

Requires [Go](https://go.dev/dl/) 1.25+ and macOS.

```sh
git clone https://github.com/niespodd/kalimate.git
cd kalimate
go run .
```

Or build a standalone binary:

```sh
go build -o kalimate .
./kalimate
```

The first launch asks for microphone permission.

## How to use it

1. **Pick a key** — tap a tine on the kalimba map, pluck any tine and Kalimate
   jumps to it, or step with **← / →**.
2. **Pluck it** — the big readout shows the note and how many cents sharp or flat
   you are; the advice line tells you to push or pull the tine.
3. **Dial it in** — adjust until it reads green (within ±5 cents). The key locks
   in automatically.
4. **Next** — press **Space / Enter** or tap **Next** to move to the next key,
   and watch the "*N of 17 tuned*" counter climb.

## How the measurement works

A plucked tine can't be trusted on a single reading, so the pluck tracker
(`internal/pitch/tracker.go`) does this per note:

1. **Overlapping analysis** — a 2048-sample YIN window advances every 1024
   samples (~43 readings/sec) for real temporal resolution.
2. **Onset detection** — an adaptive noise floor with hysteresis treats each tap
   as a distinct pluck event.
3. **Attack rejection** — the first ~90 ms (the transient) is discarded.
4. **Median estimate** — the remaining sustain samples are octave-folded to undo
   YIN's occasional octave jumps, then collapsed to one frequency by a plain
   median (deliberately not amplitude-weighted, which would bias toward the sharp
   attack).
5. **Stable, live reading** — the display shows that per-pluck frequency: steady
   for the length of a pluck, refreshed on the next one so a tine you just
   adjusted updates right away. A key is *confirmed* once its pitch holds within
   ±5 cents continuously for ~400 ms.

## Project layout

The code is split by concern — display, calculate, interface devices:

- **`internal/audio`** — *interfaces with devices*. Enumerates capture inputs and
  streams raw mono samples (CoreAudio via [malgo](https://github.com/gen2brain/malgo)).
- **`internal/pitch`** — *calculates*. Pitch detection, note mapping, and the
  pluck tracker that turns noisy frames into a stable reading.
- **`internal/ui`** — *displays*. The window, tuning meter, and kalimba map. Takes
  a plain `TunerState` struct; knows nothing about audio or math. Translations
  live in `internal/ui/translations/`.
- **`main.go`** — the controller that wires the three together and owns the device
  lifecycle.

## Tuning the tuner

Knobs live in `internal/pitch` — detection thresholds in `detect.go`
(`yinThreshold`, `rmsGate`) and pluck behavior in `tracker.go` (`neededPlucks`,
`attackMs`, onset multipliers).

## Localization

UI text is keyed in `internal/ui/translations/en.json` and resolved through
[`fyne.io/fyne/v2/lang`](https://pkg.go.dev/fyne.io/fyne/v2/lang). To add a
language, drop a `<lang>.json` (e.g. `pl.json`) next to `en.json` using the same
keys — it loads automatically on matching systems.

## License

Kalimate is released under the [MIT License](LICENSE.md). All bundled
dependencies are under permissive licenses (MIT / BSD / ISC / Apache-2.0 /
public domain) — see [LICENSE.md](LICENSE.md) for the full third-party inventory.
