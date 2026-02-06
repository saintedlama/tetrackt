# Tetrackt

The terminal music tracker for chiptune and retro-style music with a great typo.

> Attention: Heavy WIP! Nothing will work as expected!

## Features

### Arpeggiator Effect (NEW!)

Tetrackt now supports classic tracker-style arpeggiator effects! Create rapid arpeggios to simulate chords on monophonic channels, just like the retro trackers.

#### How to Use

1. Navigate to the effect column by pressing **Tab** (cycles through Note → Volume → Effect)
2. Enter a hex effect code in format `0xy` where:
   - `0` = arpeggiator effect code
   - `x` = first semitone offset (0-F hex = 0-15 semitones)
   - `y` = second semitone offset (0-F hex = 0-15 semitones)

#### Common Arpeggio Patterns

| Effect | Name          | Notes                        | Use Case             |
|--------|---------------|------------------------------|----------------------|
| `0C7`  | Major Chord   | Root, +12 (octave), +7 (5th) | Classic major sound  |
| `0C3`  | Minor Chord   | Root, +12 (octave), +3 (m3)  | Minor chord feel     |
| `047`  | Perfect 5th   | Root, +4 (maj3), +7 (5th)    | Major triad          |
| `037`  | Minor Triad   | Root, +3 (min3), +7 (5th)    | Minor triad          |
| `0CA`  | Dominant 7th  | Root, +12 (octave), +10 (7th)| Jazzy/blues          |
| `000`  | No Effect     | Disable arpeggio             | Normal playback      |

#### Navigation

- **Tab**: Move to next column (Note → Volume → Effect)
- **Shift+Tab**: Move to previous column
- **0-9, A-F**: Enter hex digits in effect column

#### Example

Load the demo file `arpeggio_demo.yaml` to hear arpeggios in action!

```bash
./tetrackt
# Press 'l' to load file
# Type: arpeggio_demo
# Press 'space' to play
```

## Controls

- **Space**: Play/Stop pattern
- **Tab/Shift+Tab**: Switch between columns
- **Arrow Keys**: Navigate pattern
- **1-7**: Enter notes (C-B)
- **+/-**: Change octave
- **Delete**: Clear note
- **s**: Save file
- **l**: Load file
- **q**: Quit
