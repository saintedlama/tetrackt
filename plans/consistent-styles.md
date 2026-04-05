# Consistent Styles

> Status: **Done**
To ensure a cohesive and visually appealing user interface, we need to establish consistent styles across the application. This includes defining a color palette and extracting additional visual components.

## Color Palette

The palette is derived from synthesizer UIs.

### Neutral / Background Scale

| Name           | Hex       | Usage in synthesizer UIs                           |
| -------------- | --------- | -------------------------------------------------- |
| `gray-darkest` | `#1c1c1e` | Main application background                        |
| `gray-dark`    | `#242428` | Panel / section backgrounds                        |
| `gray-medium`  | `#3c3c3c` | Knob bodies, control surfaces, inactive separators |
| `gray`         | `#5a5a5a` | Inactive / disabled element fill                   |
| `gray-light`   | `#8a8a8a` | Secondary labels, inactive track lanes             |
| `gray-lighter` | `#c8c8c8` | Primary labels, parameter names, readout text      |

### Accent Colors

| Name     | Hex       | Usage in synthesizer UIs                                        |
| -------- | --------- | --------------------------------------------------------------- |
| `cyan`   | `#00d4e8` | Primary accent — active elements, waveform displays, selections |
| `orange` | `#ff7700` | Envelope curves (VCA/ENV), active knob rings, highlights        |
| `purple` | `#8b2fc9` | Modulation / Random section indicators, mod wheel lane          |
| `green`  | `#00c853` | Function generator section, positive modulation arcs            |
| `pink`   | `#e81e8c` | Keyboard / velocity lane, some macro rings                      |
| `yellow` | `#ffb300` | LFO waveform fill, sequencer step accents                       |

### Semantic Aliases (intended use in Tetrackt)

| Alias               | Mapped color   | Purpose                                     |
| ------------------- | -------------- | ------------------------------------------- |
| `background`        | `gray-darkest` | Root terminal background                    |
| `surface`           | `gray-dark`    | Panel / bordered section background         |
| `border`            | `gray-medium`  | Inactive panel border                       |
| `border-active`     | `cyan`         | Active / focused panel border               |
| `text`              | `gray-lighter` | Default readable text                       |
| `text-muted`        | `gray-light`   | Labels, secondary info                      |
| `text-disabled`     | `gray`         | Disabled / empty cells                      |
| `accent-primary`    | `cyan`         | Selected rows, cursor cells, active modes   |
| `accent-envelope`   | `orange`       | Envelope editor highlights                  |
| `accent-oscillator` | `cyan`         | Oscillator waveform type indicator          |
| `accent-modulation` | `purple`       | Modulation / LFO indicators                 |
| `accent-play`       | `green`        | Playback row highlight                      |
| `accent-instrument` | `pink`         | Instrument / preset selection highlight     |
| `accent-warning`    | `yellow`       | Errors, warnings, unsaved-changes indicator |

## Implementation Plan

1. Define the color palette in a central location (e.g., `ui/colors.go`) as constants or variables.
2. Refactor existing components to use the new color aliases instead of hardcoded hex values.
3. Extract common visual components (e.g., panel borders, separators) into reusable functions or types that apply the consistent styles.
4. Ensure that all new UI elements adhere to the established color palette and style guidelines.
5. Update documentation and comments to reference the new color aliases and style conventions.
