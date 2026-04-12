# TeTrackT

TeTrackT is a terminal music tracker for chiptune and retro-style music.

## Installing

Install prebuilt binaries from GitHub Releases:

- Go to releases page: <https://github.com/saintedlama/tetrackt/releases>
- Download the archive for your platform.
- Extract it to a directory of your choice.
- Run the executable from that directory.

If a release for your platform is not available yet, build from source:

```sh
go build .
```

Built-in songs included:

- Press `l` to open the load dialog, then press `Enter` on "Quickstart" or "Demo Song".
- Optional file-based module: `modules/quickstart.json`
- Full chiptune demo module: `modules/chiptune-demo.json`
- You can also press `l` to open the load dialog and select a `.json` module.

## Features

- Tracker-focused TUI workflow built with Bubble Tea and Lip Gloss.
- Integrated synthesizer engine with per-track synth settings.
- Oscillator and tone-shaping features:
  - multiple oscillator waveforms
  - wavetable support
  - pulse-width control
  - LFSR/noise options
- Modulation and dynamics:
  - ADSR envelope controls
  - LFO modulation (pitch, volume, cutoff, pulse width)
  - filter and filter envelope controls
- Performance and sequencing tools:
  - arpeggio, portamento, detune, and gate behaviors
  - effect processing and per-track mixer support
  - speed and tracker UX improvements (navigation/editing helpers)
- Presets and persistence:
  - synth patch bank presets
  - save/load module data via JSON

## Contributing

Contributions are welcome.

- Fork the repo and create a feature branch.
- Make crazy changes.
- Open a pull request
