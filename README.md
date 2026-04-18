# Tetrackt

Tetrackt is a noise maker for the command line. A creative playground for whisking crazy sounds together — not for making great music, but for making interesting ones. It is not a chiptune tracker, a sequencer, or a DAW. It is a noise maker.

## What's inside

Three oscillators per patch. Detune them against each other, run them through a filter, slap an LFO on the cutoff, and watch something weird happen. That's the idea.

- Three oscillators: sine, square, triangle, saw, reverse saw, noise, periodic noise, and wavetable
- Band-limited wavetable tones for soft saw, organ, glass, bass, strings, flute, brass, chime, and voice-like stuff
- Per-oscillator ADSR envelopes
- LFO routing to pitch, volume, cutoff, pulse width, and detune
- Filter with low-pass, high-pass, and band-pass modes plus its own envelope
- Portamento/glide
- Per-oscillator volume and pan, plus master volume
- Mix character: linear summing, NES-style nonlinear pulse mix, or soft clipping for dirtier tones

## Patch bank

Hit `b` on the synth screen to open the patch browser. It ships with a bunch of presets grouped into categories:

- Lead
- Bass
- Pad
- Arp
- Percussive
- SFX

NES/Game Boy/C64 flavors, arcade tones, analog-style pads and basses, 808 and 909 style hits. Start from one of those and break it until it sounds wrong in a good way.

You can filter by category or tag, save your own patches, rename them, delete them. Custom patches live in `~/.tetrackt`.

## The tracker

Tetrackt has a tracker grid for sequencing. It's not the point — the noise is the point — but having a grid means you can loop a pattern and keep tweaking the synth while it runs.

Two modes: `EDIT` for writing notes, `NAV` for moving around. Switch with `Space`.

### Note input

Default layout is QWERTY. Want QWERTZ? Put this in `~/.tetrackt`:

```json
{
  "inputProfile": "qwertz"
}
```

| Row | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Upper keys | `Q` | `2` | `W` | `3` | `E` | `R` | `5` | `T` | `6` | `Y` | `7` | `U` |
| Upper notes | `C` | `C#` | `D` | `D#` | `E` | `F` | `F#` | `G` | `G#` | `A` | `A#` | `B` |
| Lower keys | `Z` | `S` | `X` | `D` | `C` | `V` | `G` | `B` | `H` | `N` | `J` | `M` |
| Lower notes | `C` | `C#` | `D` | `D#` | `E` | `F` | `F#` | `G` | `G#` | `A` | `A#` | `B` |

### Moving around

- `Arrow keys`: move
- `PgUp` / `PgDn`: jump by viewport height
- `Home` / `End`: first/last row
- `Tab` / `Shift+Tab`: move between subcolumns
- `Delete`: clear value
- `Insert`: insert space in current track
- `Shift+Insert`: insert row across all tracks

### Selection and editing

- `Shift+Arrow`: rectangular selection
- `Alt+C` / `Alt+X` / `Alt+V`: copy, cut, paste
- `Alt+Shift+V`: paste effects only, keep destination notes
- `Alt+Up` / `Alt+Down`: transpose by semitone
- `Alt+Shift+Up` / `Alt+Shift+Down`: transpose by octave
- `F8` / `F7` and `Shift+F8` / `Shift+F7`: transpose aliases

### Effects

Effects go in the effect column. Accepts `0..7` or letter aliases.

- `0xx`: no effect
- `1xy` or `V`: vibrato
- `2xx` or `S`: volume slide
- `3xx` or `C`: note cut
- `4xx` or `D`: note delay
- `5xx` or `T`: per-row tick override
- `6xx` or `O`: continuous note toggle
- `7xy` or `A`: arpeggio preset

Arpeggio presets: up, down, converge, diverge, random.

## Screens and playback

Two screens: **Tracker** and **Synth**. Switch with `Ctrl+t`.

- `Ctrl+Left` / `Ctrl+Right`: move between tracker panels
- `Ctrl+Arrow keys`: move between synth panels
- `p`: play/pause from row 0
- `P`: loop up to the current row
- `?`: key reference

When you enter a note it previews through the current patch, so you hear changes immediately.

## Save, load, export

- `Ctrl+s`: save
- `Ctrl+l`: load

Saves as JSON. You can also export directly to `.wav` from the save dialog — same render engine as live playback, so all the effects and modulation make it to the file.

## First run

Tetrackt ships with a couple of modules to poke around in.

- `Ctrl+l` → load `Quickstart` for a quick orientation
- `Ctrl+l` → load `Demo Song` to hear what it can do
- Or load `modules/quickstart.json` / `modules/chiptune-demo.json` directly

## Install

Grab a binary from GitHub Releases:

- <https://github.com/saintedlama/tetrackt/releases>

Or build it:

```sh
go build .
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

New tracker trick, meaner bass patch, cleaner render path, more cursed sound effect — pull requests are welcome.
