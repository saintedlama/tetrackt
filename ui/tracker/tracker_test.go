package tracker

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
)

func TestSpaceTogglesEditMode(t *testing.T) {
	m := NewTracker(2, 8, 80, 24)
	if m.Mode != NavigateMode {
		t.Fatalf("expected default navigate mode, got %v", m.Mode)
	}

	_, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if m.Mode != EditMode {
		t.Fatalf("expected edit mode after space, got %v", m.Mode)
	}
}

func TestNoteEntryInEditModeEmitsPreviewMessage(t *testing.T) {
	m := NewTracker(2, 8, 80, 24)
	m.Mode = EditMode
	m.Octave = 4
	m.CursorCol = ColumnNote

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
	m.Mode = EditMode
	m.Octave = 4
	m.EditStep = 2
	m.CursorCol = ColumnNote

	_, _ = m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if m.CursorRow != 2 {
		t.Fatalf("expected cursor row 2, got %d", m.CursorRow)
	}
}

func TestTabMovesToNextSubcolumn(t *testing.T) {
	m := NewTracker(2, 8, 80, 24)
	if m.CursorCol != ColumnNote {
		t.Fatalf("expected initial note column, got %v", m.CursorCol)
	}

	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.CursorCol != ColumnVolume {
		t.Fatalf("expected volume column after tab, got %v", m.CursorCol)
	}
}

func TestInsertTrackSpaceShiftsRowsDown(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.CursorTrack = 0
	m.CursorRow = 1
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
	m.CursorTrack = 0
	m.CursorRow = 0
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

	m.selection = trackerSelection{
		Active:      true,
		AnchorRow:   0,
		AnchorTrack: 0,
		EndRow:      1,
		EndTrack:    1,
	}

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
