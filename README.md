# Tetrackt

Tetrackt is a terminal tracker for people who like hex, pulse waves, acid squelch, FM-adjacent weirdness, and the moment a tiny pattern suddenly turns into a whole soundtrack.

It gives you a fast tracker workflow on one side and a proper built-in synth engine on the other: per-track instruments, three oscillators, envelopes, filters, LFO routing, patch browsing, live playback, and WAV export without leaving the app.

## Why it rips

- Tracker-first workflow with fast keyboard editing, block selection, transpose, copy/cut/paste, and inline effect entry.
- Per-track synth design instead of fixed playback samples.
- Three-oscillator synth engine with pulse width, detune, wavetable tones, noise, periodic noise, envelopes, and portamento.
- Modulation that can target pitch, volume, cutoff, pulse width, and detune.
- Filter section with low-pass, high-pass, and band-pass modes plus filter envelope support.
- Mixer character modes including linear summing, an NES-style nonlinear pulse mix, and soft clipping for dirtier tones.
- Built-in patch bank full of leads, basses, pads, arps, drums, 808/909-style hits, and game-style SFX.
- Save songs as JSON modules or render them directly to `.wav`.

## Install

Grab a prebuilt binary from GitHub Releases:

- <https://github.com/saintedlama/tetrackt/releases>

Or build it yourself:

```sh
go build .
```

Run the resulting binary from the project directory or wherever you unpacked it.

## First boot

Tetrackt already ships with music in the box.

- Press `Ctrl+l` to open the load dialog.
- Load `Quickstart` if you want a fast orientation.
- Load `Demo Song` if you want to hear the machine stretch a bit.
- You can also load external module files such as `modules/quickstart.json` and `modules/chiptune-demo.json`.

## What is actually inside the synth

This is not a token “retro” sound toggle. The synth engine in `audio/` is a real signal path:

- up to **three oscillators per patch**
- oscillator types including **sine, square, triangle, saw, reverse saw, noise, periodic noise, and wavetable**
- **band-limited wavetable tones** for sounds like soft saw, soft square, organ, glass, bass, strings, flute, brass, chime, and voice-like spectra
- per-oscillator **ADSR envelopes**
- **LFO routing** to pitch, volume, cutoff, pulse width, and detune
- **filter modulation** via both LFO and filter envelope
- **portamento/glide**
- per-oscillator **volume and stereo pan**, plus master volume
- selectable mix character: **Linear**, **NES**, or **Clip**

That means you can build straight-up chip leads, unstable detuned pads, fake SID growls, percussive noise hits, or soft analog-ish textures without changing tools.

## Patch bank

The synth screen has a patch bank (`b`) with built-in instruments grouped into categories like:

- Lead
- Bass
- Pad
- Arp
- Percussive
- SFX

The built-in presets cover a lot of territory: NES/Game Boy/C64-flavored patches, arcade tones, analog-style pads and basses, plus drum-machine-inspired kits like 808 and 909 style hits.

The patch browser also supports:

- category filtering
- tag filtering
- custom patch saving
- custom patch rename/delete

Your custom synth patches are stored in `~/.tetrackt`.

## Tracker workflow

Tetrackt uses a two-mode tracker workflow:

- `EDIT` for writing notes and values
- `NAV` for moving around cleanly

Switch modes with `Space`.

### Note input

Default profile is `QWERTY`.

If you want `QWERTZ`, set this in `~/.tetrackt`:

```json
{
  "inputProfile": "qwertz"
}
```

Default QWERTY note layout:

| Row | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Upper keys | `Q` | `2` | `W` | `3` | `E` | `R` | `5` | `T` | `6` | `Y` | `7` | `U` |
| Upper notes | `C` | `C#` | `D` | `D#` | `E` | `F` | `F#` | `G` | `G#` | `A` | `A#` | `B` |
| Lower keys | `Z` | `S` | `X` | `D` | `C` | `V` | `G` | `B` | `H` | `N` | `J` | `M` |
| Lower notes | `C` | `C#` | `D` | `D#` | `E` | `F` | `F#` | `G` | `G#` | `A` | `A#` | `B` |

### Core controls

- `Arrow keys`: move around the grid
- `PgUp` / `PgDn`: jump by viewport height
- `Home` / `End`: first/last row
- `Tab` / `Shift+Tab`: move between subcolumns
- `Delete`: clear the focused value
- `Insert`: insert space in current track
- `Shift+Insert`: insert row space across all tracks

### Pattern editing power moves

- `Shift+Arrow`: rectangular selection
- `Alt+C` / `Alt+X` / `Alt+V`: copy, cut, paste selection
- `Alt+Shift+V`: paste effects only and keep the destination note
- `Alt+Up` / `Alt+Down`: transpose by semitone
- `Alt+Shift+Up` / `Alt+Shift+Down`: transpose by octave
- `F8` / `F7` and `Shift+F8` / `Shift+F7`: transpose aliases

### Inline tracker effects

Effects are edited directly in the grid. The effect column accepts `0..7` or letter aliases.

- `0xx`: clear / no effect
- `1xy` or `V`: vibrato
- `2xx` or `S`: volume slide
- `3xx` or `C`: note cut
- `4xx` or `D`: note delay
- `5xx` or `T`: per-row tick override
- `6xx` or `O`: continuous note toggle
- `7xy` or `A`: arpeggio preset

Arpeggio presets include up, down, converge, diverge, and random variants.

## Screens and playback

Tetrackt is split into two main screens:

- **Tracker** for sequencing
- **Synth** for instrument design

Useful keys:

- `Ctrl+t`: switch between Tracker and Synth
- `Ctrl+Left` / `Ctrl+Right`: move between tracker panels
- `Ctrl+Arrow keys`: move between synth panels
- `p`: play/pause from row 0
- `P`: loop up to the current row
- `?`: open the in-app key reference

When you enter notes, Tetrackt previews them through the current synth patch, so sound design and sequencing stay tightly connected.

## Save, load, export

- `Ctrl+s`: save
- `Ctrl+l`: load

Tetrackt supports two output paths:

- **module save/load** as JSON
- **WAV export** directly from the save dialog

The offline WAV export uses the same render engine as playback, so tracker effects, envelopes, modulation, and per-row behavior all make the trip to disk.

## Project layout

If you want to poke around in the code:

```text
audio/        synthesis engine, oscillators, envelopes, LFO, filter, mixer
render/       live playback and offline WAV rendering
ui/tracker/   tracker editor, navigation, selections, row effects
ui/synth/     synth editor, patch bank, parameter panels
persistence/  module save/load and patch bank persistence
main.go       app wiring, dialogs, playback, screen routing
```

## Development

```sh
go test ./...
go build .
go vet ./...
go run .
```

Or with Make:

```sh
make test
make build
make lint
make run
```

## Contributing

If you want to add a new tracker trick, a meaner bass patch, a cleaner render path, or a more cursed sound effect, pull requests are welcome.
