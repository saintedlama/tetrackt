package tracker

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
)

func TestSpaceTogglesEditMode(t *testing.T) {
	m := NewTracker(2, 8, 80, 24)
	if m.Mode != navigateMode {
		t.Fatalf("expected default navigate mode, got %v", m.Mode)
	}

	_, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if m.Mode != editMode {
		t.Fatalf("expected edit mode after space, got %v", m.Mode)
	}
}

func TestNoteEntryInEditModeEmitsPreviewMessage(t *testing.T) {
	m := NewTracker(2, 8, 80, 24)
	m.Mode = editMode
	m.Octave = 4
	m.CursorCol = columnNote

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if cmd == nil {
		t.Fatal("expected note entry command")
	}
	msg := cmd()
	noteMsg, ok := msg.(TrackerNoteEntered)
	if !ok {
		t.Fatalf("expected TrackerNoteEntered, got %T", msg)
	}
	if noteMsg.Note.Base != audio.Base("C") || noteMsg.Note.Octave != 4 {
		t.Fatalf("unexpected note in message: %+v", noteMsg.Note)
	}
	if m.Tracks[0].Rows[0].Note.Base != audio.Base("C") {
		t.Fatalf("expected C note in current cell")
	}
}

func TestEditStepAdvancesCursor(t *testing.T) {
	m := NewTracker(2, 16, 80, 24)
	m.Mode = editMode
	m.Octave = 4
	m.EditStep = 2
	m.CursorCol = columnNote

	_, _ = m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if m.nav.CursorRow() != 2 {
		t.Fatalf("expected cursor row 2, got %d", m.nav.CursorRow())
	}
}

func TestTabMovesToNextSubcolumn(t *testing.T) {
	m := NewTracker(2, 8, 80, 24)
	if m.CursorCol != columnNote {
		t.Fatalf("expected initial note column, got %v", m.CursorCol)
	}

	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.CursorCol != columnVolume {
		t.Fatalf("expected volume column after tab, got %v", m.CursorCol)
	}
}

func TestInsertTrackSpaceShiftsRowsDown(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.nav.SetCursorPosition(0, 1)
	m.Tracks[0].Rows[1].Note = audio.Note{Base: audio.Base("C"), Octave: 4}
	m.Tracks[0].Rows[2].Note = audio.Note{Base: audio.Base("D"), Octave: 4}

	edited := m.handleGlobalEditingKey("insert")
	if !edited {
		t.Fatal("expected insert to be treated as edit")
	}

	if got := m.Tracks[0].Rows[1].Note.Base; got != audio.BaseOff {
		t.Fatalf("expected inserted empty row at cursor, got %q", got)
	}
	if got := m.Tracks[0].Rows[2].Note.Base; got != audio.Base("C") {
		t.Fatalf("expected previous row shifted down to row 2, got %q", got)
	}
}

func TestTransposeCurrentStoredNoteSemitone(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.nav.SetCursorPosition(0, 0)
	m.Tracks[0].Rows[0].Note = audio.Note{Base: audio.BaseC, Octave: 4}

	edited := m.handleGlobalEditingKey("alt+up")
	if !edited {
		t.Fatal("expected transpose to edit current note")
	}

	got := m.Tracks[0].Rows[0].Note
	if got.Base != audio.BaseCs || got.Octave != 4 {
		t.Fatalf("expected C#4 after transpose, got %+v", got)
	}
}

func TestTransposeSelectedNotesOctave(t *testing.T) {
	m := NewTracker(2, 4, 80, 24)
	m.Tracks[0].Rows[0].Note = audio.Note{Base: audio.BaseC, Octave: 4}
	m.Tracks[1].Rows[1].Note = audio.Note{Base: audio.BaseE, Octave: 4}

	// Use navigation grid to set up selection
	m.nav.SetCursorPosition(0, 0)
	m.nav.MoveExtending(1, 1)

	edited := m.handleGlobalEditingKey("alt+shift+up")
	if !edited {
		t.Fatal("expected transpose to edit selected notes")
	}

	n0 := m.Tracks[0].Rows[0].Note
	if n0.Base != audio.BaseC || n0.Octave != 5 {
		t.Fatalf("expected C5, got %+v", n0)
	}
	n1 := m.Tracks[1].Rows[1].Note
	if n1.Base != audio.BaseE || n1.Octave != 5 {
		t.Fatalf("expected E5, got %+v", n1)
	}
}

func TestParseEffectCommandNibbleSupportsInlineExtensions(t *testing.T) {
	for i := 0; i <= int(EffectArpPreset); i++ {
		typ, ok := parseEffectCommandNibble(i)
		if !ok {
			t.Fatalf("expected nibble %d to be valid", i)
		}
		if int(typ) != i {
			t.Fatalf("expected type %d, got %d", i, typ)
		}
	}

	if _, ok := parseEffectCommandNibble(8); ok {
		t.Fatal("expected nibble 8 to be invalid")
	}
}

func TestParseEffectCommandKeyAliases(t *testing.T) {
	cases := map[string]EffectType{
		"v": EffectVibrato,
		"s": EffectVolumeSlide,
		"c": EffectNoteCut,
		"d": EffectNoteDelay,
		"t": EffectRowTicks,
		"o": EffectContinuous,
		"a": EffectArpPreset,
	}

	for key, want := range cases {
		got, ok := parseEffectCommandKey(key)
		if !ok || got != want {
			t.Fatalf("key %q expected %v, got %v (ok=%v)", key, want, got, ok)
		}
	}
}

func TestInlineRowTicksCommand(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.Mode = editMode
	m.CursorCol = columnEffect

	_, _ = m.Update(tea.KeyPressMsg{Text: "5"})
	if got := m.Tracks[0].Rows[0].Effect.Type; got != EffectRowTicks {
		t.Fatalf("expected EffectRowTicks, got %v", got)
	}

	m.CursorCol = columnParam
	_, _ = m.Update(tea.KeyPressMsg{Text: "0"})
	_, _ = m.Update(tea.KeyPressMsg{Text: "C"})

	row := m.Tracks[0].Rows[0]
	if row.Ticks != 12 {
		t.Fatalf("expected ticks=12, got %d", row.Ticks)
	}
	if row.Effect.Param != 0x0C {
		t.Fatalf("expected effect param 0x0C, got %#x", row.Effect.Param)
	}
}

func TestInlineContinuousCommandToggle(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.Mode = editMode
	m.CursorCol = columnEffect

	_, _ = m.Update(tea.KeyPressMsg{Text: "6"})
	m.CursorCol = columnParam
	_, _ = m.Update(tea.KeyPressMsg{Text: "0"})
	_, _ = m.Update(tea.KeyPressMsg{Text: "1"})

	if !m.Tracks[0].Rows[0].Continuous {
		t.Fatal("expected continuous to be enabled")
	}

	m.nav.Move(0, 1)
	m.CursorCol = columnEffect
	_, _ = m.Update(tea.KeyPressMsg{Text: "6"})
	m.CursorCol = columnParam
	_, _ = m.Update(tea.KeyPressMsg{Text: "0"})
	_, _ = m.Update(tea.KeyPressMsg{Text: "0"})

	if m.Tracks[0].Rows[1].Continuous {
		t.Fatal("expected continuous to be disabled")
	}
}

func TestInlineArpPresetCommand(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.Mode = editMode
	m.CursorCol = columnEffect
	_, _ = m.Update(tea.KeyPressMsg{Text: "7"})

	m.CursorCol = columnParam
	_, _ = m.Update(tea.KeyPressMsg{Text: "1"})
	_, _ = m.Update(tea.KeyPressMsg{Text: "4"})

	row := m.Tracks[0].Rows[0]
	if row.Effect.Type != EffectArpPreset {
		t.Fatalf("expected EffectArpPreset, got %v", row.Effect.Type)
	}
	if !row.Arpeggio.IsActive() {
		t.Fatal("expected arp offsets to be generated")
	}
	if !row.Continuous {
		t.Fatal("expected arp preset command to force continuous")
	}
}

func TestTransposeAltShiftOrderVariants(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.nav.SetCursorPosition(0, 0)
	m.Tracks[0].Rows[0].Note = audio.Note{Base: audio.BaseC, Octave: 4}

	edited := m.handleGlobalEditingKey("shift+alt+up")
	if !edited {
		t.Fatal("expected shift+alt+up to transpose")
	}

	got := m.Tracks[0].Rows[0].Note
	if got.Base != audio.BaseC || got.Octave != 5 {
		t.Fatalf("expected C5 after shift+alt+up, got %+v", got)
	}

	edited = m.handleGlobalEditingKey("shift+alt+down")
	if !edited {
		t.Fatal("expected shift+alt+down to transpose")
	}

	got = m.Tracks[0].Rows[0].Note
	if got.Base != audio.BaseC || got.Octave != 4 {
		t.Fatalf("expected C4 after shift+alt+down, got %+v", got)
	}
}

func TestPasteEffectsOnlyKeepsDestinationNote(t *testing.T) {
	m := NewTracker(2, 4, 80, 24)

	// Source cell contains note + effects payload.
	m.Tracks[0].Rows[0] = TrackRow{
		Note:       audio.Note{Base: audio.BaseE, Octave: 4},
		Volume:     48,
		Ticks:      12,
		Continuous: true,
		Arpeggio:   audio.ArpeggioEffect{Offsets: []int{0, 4, 7}},
		Effect:     TrackerEffect{Type: EffectVibrato, Param: 0x24},
	}

	// Copy source cell.
	m.nav.SetCursorPosition(0, 0)
	m.nav.ClearSelection()
	m.handleGlobalEditingKey("alt+c")

	// Destination has a different note that should be preserved.
	m.Tracks[1].Rows[1] = TrackRow{Note: audio.Note{Base: audio.BaseA, Octave: 5}}
	m.nav.SetCursorPosition(1, 1)

	edited := m.handleGlobalEditingKey("alt+shift+v")
	if !edited {
		t.Fatal("expected alt+shift+v to edit destination cell")
	}

	got := m.Tracks[1].Rows[1]
	if got.Note.Base != audio.BaseA || got.Note.Octave != 5 {
		t.Fatalf("expected destination note A5 to be preserved, got %+v", got.Note)
	}
	if got.Volume != 48 || got.Ticks != 12 || !got.Continuous {
		t.Fatalf("expected effects payload copied, got row %+v", got)
	}
	if !got.Arpeggio.IsActive() || got.Effect.Type != EffectVibrato || got.Effect.Param != 0x24 {
		t.Fatalf("expected arp/effect copied, got row %+v", got)
	}
}
