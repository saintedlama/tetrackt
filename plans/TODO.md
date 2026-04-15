# Tetrackt — Active Plans

| Plan                              | Description                                                                              |
| --------------------------------- | ---------------------------------------------------------------------------------------- |
| `audio-backend.md`                | Switch from `beep` to `oto` for more reliable, glitch-free audio playback                |
| `screenshots-with-vhs.md`         | Add README screenshots using the `vhs` tool (Charmbracelet)                              |
| `tracker-ux-2-cell-indicators.md` | 4th column in grid cells showing per-row Ticks/Continuous state (`T4`, `C.`, etc.)       |
| `tracker-ux-3-volume-editing.md`  | Inline per-row volume editing via sub-cursor (`V` key), two-digit decimal entry          |
| `tracker-ux-4-page-navigation.md` | `PgUp`/`PgDn` keys for full-page jumps in the pattern editor                             |
| `tracker-ux-5-speed-setting.md`   | Speed (sub-ticks per row) row in the Settings panel                                      |
| `tracker-ux-6-row-markers.md`     | Yellow accent on row numbers that have non-default Ticks, Continuous, or Arpeggio        |
| `tracker-ux-7-copy-paste.md`      | Unified clipboard: copy/paste cell, row, track, synth; selection with anchor; fill/clear |

## Notes

- `tracker-ux-2-cell-indicators.md` predates the inline effects implementation (`tracker-ux-8`). The cell column layout described in that plan may need revisiting now that inline effects editing is in place, since the column structure has changed.
- `audio-backend.md` (`beep` → `oto`) is the most infrastructural open item — it could affect all audio work.
- `screenshots-with-vhs.md` is purely a documentation/tooling task.
