# Tetrackt

Tetrackt is a music tracker for chiptune and retro-style music built for the terminal. It allows users to create, edit, and play music using a text-based interface, reminiscent of classic music trackers from the 1980s and 1990s.

## Technologies Used

- Go: The application is developed in Go, leveraging its performance and concurrency features.
- Terminal UI Libraries: bubbletea, bubbles, lipgloss and ntcharts are used to create an interactive and visually appealing terminal interface.
- Synthetic Audio Generation: The application includes functionality for generating chiptune-style audio directly within the terminal environment using the Gosound library.
- Cross-Platform Compatibility: Tetrackt is designed to work across various operating systems, including Windows, macOS, and Linux.

## Features

### Instrument Editor
- Create and modify instruments with various waveforms (square, triangle, sawtooth, noise).
- Adjust ADSR envelope settings for dynamic sound shaping.
- Save and load custom instruments for reuse in different projects.

### Pattern Editor
- Compose music using a grid-based pattern editor.
- Input notes, effects, and commands in a familiar tracker format.
- Navigate patterns using keyboard shortcuts for efficient editing.

### Song Arrangement
- Arrange patterns into complete songs using a sequence editor.
- Manage multiple patterns and their order within a song.
- Support for looping and pattern chaining.
