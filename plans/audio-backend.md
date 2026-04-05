# Audio Backend

Currently the audio backend is implemented using the `beep` library. As we're experiencing some "glitches" in the audio output, we might want to consider switching to `oto` for more reliable audio playback.

The goal is to keep key concepts of beep, such as the `Streamer` interface, while leveraging oto's capabilities for smoother audio output.
