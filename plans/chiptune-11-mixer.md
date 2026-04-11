# Chiptune Mixer Enhancements

**Status:** Done

## Current state

`audio.Mixer` performs additive mixing of two oscillator channels with independent per-channel volume (`Volume1`, `Volume2`). The `Mix(s1, s2)` method applies `effects.Volume` to each streamer then sums them with `beep.Mix`.

## Missing capabilities (priority order)

### 1. Stereo panning per channel

Hard-panning is a defining trait of chips like the AY-3-8910 and Game Boy DMG. Both channels currently contribute equally to both stereo sides.

- Add a `Pan` field per channel: `-1.0` (full left) → `0.0` (centre) → `+1.0` (full right).
- Implementation: after applying volume, split the mono signal into L/R by scaling each side proportionally (e.g. constant-power pan law: `L = cos(θ)`, `R = sin(θ)` where `θ = (pan+1)*π/4`).

### 2. Master volume

A single output gain applied after all channels are summed. Real chips exposed this as a global register (NES master volume, SID volume register). Useful for song-level fades and loudness control independent of per-channel balance.

- Add a `MasterVolume float64` field to `Mixer`.
- Apply a final `effects.Volume` stage after `beep.Mix`.

### 3. N-channel mixing

The current design is hardcoded to two oscillators. Real chips have more:

| Chip      | Channels                          |
| --------- | --------------------------------- |
| NES APU   | 2 pulse + triangle + noise + DPCM |
| SID       | 3                                 |
| AY-3-8910 | 3                                 |
| GB DMG    | 4 (2 pulse + wave + noise)        |

Generalise to a slice of `ChannelConfig{Volume, Pan}` and a `MixAll(streamers ...beep.Streamer)` method.

### 4. Channel mute

Chips allowed silencing individual channels via a mixer register bit (AY-3-8910 mixer register is the canonical example) without zeroing the stored volume value. Preserving the volume while toggling output is how live chip musicians work.

- Add a `Mute1`, `Mute2` bool (or a `Muted bool` field on a `ChannelConfig` struct).
- Muted channels emit silence while retaining their `Volume` and `Pan` values.

### 5. Non-linear summing (authenticity / optional)

The NES APU sums pulse channels through a lookup table rather than linear addition, producing a characteristic slightly-crunchy blend. The SID similarly clips under heavy load. This is optional but contributes to the "that sound" quality.

- Could be modelled as a post-mix soft-clip or a table-driven combiner gated behind a `MixMode` enum (`Linear`, `NESPulse`, `SoftClip`).

## Implementation notes

- Steps 1–4 are additive and can be done incrementally without breaking `Synth`.
- The `ui/synth/mixer.go` panel will need updating for each new field (pan bars, mute toggles, master volume knob).
- Persistence (`SavedSong` / JSON) will need new fields; add with zero-value defaults to keep existing song files loading correctly.
