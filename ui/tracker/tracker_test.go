package tracker

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/notes"
	"github.com/tetrackt/tetrackt/ui/tracker/effects"
)

func TestNoteEntryInEditModeEmitsPreviewMessage(t *testing.T) {
	m := NewTracker(2, 8, 80, 24)
	m.Octave = 4
	m.CursorCol = columnNote

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	require.NotNil(t, cmd, "expected note entry command")
	msg := cmd()
	noteMsg, ok := msg.(TrackerNoteEntered)
	require.True(t, ok, "expected TrackerNoteEntered, got %T", msg)
	assert.Equal(t, notes.Base("C"), noteMsg.Note.Base, "unexpected note base")
	assert.Equal(t, notes.Octave(4), noteMsg.Note.Octave, "unexpected note octave")
	assert.Equal(t, notes.Base("C"), m.Tracks[0].Rows[0].Note.Base, "expected C note in current cell")
}

func TestEditStepAdvancesCursor(t *testing.T) {
	m := NewTracker(2, 16, 80, 24)
	m.Octave = 4
	m.EditStep = 2
	m.CursorCol = columnNote

	_, _ = m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	assert.Equal(t, 2, m.nav.CursorRow(), "expected cursor row 2")
}

func TestTabCyclesThroughColumnsAndTracks(t *testing.T) {
	m := NewTracker(2, 8, 80, 24)
	require.Equal(t, columnNote, m.CursorCol, "expected initial note column")
	require.Equal(t, 0, m.nav.CursorTrack(), "expected initial track 0")

	// Tab 1: NOTE → VOL on track 0
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	assert.Equal(t, columnVolume, m.CursorCol, "expected volume column after first tab")
	assert.Equal(t, 0, m.nav.CursorTrack(), "expected track 0 after first tab")

	// Tab 2: VOL → FX on track 0
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	assert.Equal(t, columnFX, m.CursorCol, "expected FX column after second tab")
	assert.Equal(t, 0, m.nav.CursorTrack(), "expected track 0 after second tab")

	// Tab 3: FX → NOTE on track 1
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	assert.Equal(t, columnNote, m.CursorCol, "expected note column after third tab")
	assert.Equal(t, 1, m.nav.CursorTrack(), "expected track 1 after third tab")
}

func TestInsertTrackSpaceShiftsRowsDown(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.nav.SetCursorPosition(0, 1)
	m.Tracks[0].Rows[1].Note = notes.Note{Base: notes.BaseC, Octave: 4}
	m.Tracks[0].Rows[2].Note = notes.Note{Base: notes.Base("D"), Octave: 4}

	edited := m.handleGlobalEditingKey("insert")
	require.True(t, edited, "expected insert to be treated as edit")

	assert.Equal(t, notes.BaseOff, m.Tracks[0].Rows[1].Note.Base, "expected inserted empty row at cursor")
	assert.Equal(t, notes.Base("C"), m.Tracks[0].Rows[2].Note.Base, "expected previous row shifted down to row 2")
}

func TestTransposeCurrentStoredNoteSemitone(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.nav.SetCursorPosition(0, 0)
	m.Tracks[0].Rows[0].Note = notes.Note{Base: notes.BaseC, Octave: 4}

	edited := m.handleGlobalEditingKey("alt+up")
	require.True(t, edited, "expected transpose to edit current note")

	got := m.Tracks[0].Rows[0].Note
	assert.Equal(t, notes.BaseCs, got.Base, "expected C# after transpose")
	assert.Equal(t, notes.Octave(4), got.Octave, "expected octave 4 after transpose")
}

func TestTransposeSelectedNotesOctave(t *testing.T) {
	m := NewTracker(2, 4, 80, 24)
	m.Tracks[0].Rows[0].Note = notes.Note{Base: notes.BaseC, Octave: 4}
	m.Tracks[1].Rows[1].Note = notes.Note{Base: notes.BaseE, Octave: 4}

	// Use navigation grid to set up selection
	m.nav.SetCursorPosition(0, 0)
	m.nav.MoveExtending(1, 1)

	edited := m.handleGlobalEditingKey("alt+shift+up")
	require.True(t, edited, "expected transpose to edit selected notes")

	n0 := m.Tracks[0].Rows[0].Note
	assert.Equal(t, notes.BaseC, n0.Base, "expected C")
	assert.Equal(t, notes.Octave(5), n0.Octave, "expected C5")

	n1 := m.Tracks[1].Rows[1].Note
	assert.Equal(t, notes.BaseE, n1.Base, "expected E")
	assert.Equal(t, notes.Octave(5), n1.Octave, "expected E5")
}

func TestParseEffectCommandNibbleSupportsInlineExtensions(t *testing.T) {
	for i := 0; i <= int(EffectArpPreset); i++ {
		typ, ok := effects.ParseNibble(i)
		require.True(t, ok, "expected nibble %d to be valid", i)
		assert.Equal(t, i, int(typ), "expected type %d", i)
	}

	_, ok := effects.ParseNibble(8)
	assert.False(t, ok, "expected nibble 8 to be invalid")
}

func TestParseEffectCommandKeyAliases(t *testing.T) {
	cases := map[string]EffectType{
		"v": EffectVibrato,
		"s": EffectVolumeSlide,
		"c": EffectNoteCut,
		"d": EffectNoteDelay,
		"t": EffectRowTicks,
		"a": EffectArpPreset,
	}

	for key, want := range cases {
		got, ok := effects.ParseKey(key)
		require.True(t, ok, "key %q: expected ok=true", key)
		assert.Equal(t, want, EffectType(got), "key %q expected %v", key, want)
	}
}

func TestTransposeAltShiftOrderVariants(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.nav.SetCursorPosition(0, 0)
	m.Tracks[0].Rows[0].Note = notes.Note{Base: notes.BaseC, Octave: 4}

	edited := m.handleGlobalEditingKey("shift+alt+up")
	require.True(t, edited, "expected shift+alt+up to transpose")

	got := m.Tracks[0].Rows[0].Note
	assert.Equal(t, notes.BaseC, got.Base, "expected C")
	assert.Equal(t, notes.Octave(5), got.Octave, "expected octave 5 after shift+alt+up")

	edited = m.handleGlobalEditingKey("shift+alt+down")
	require.True(t, edited, "expected shift+alt+down to transpose")

	got = m.Tracks[0].Rows[0].Note
	assert.Equal(t, notes.BaseC, got.Base, "expected C")
	assert.Equal(t, notes.Octave(4), got.Octave, "expected octave 4 after shift+alt+down")
}

func TestPasteEffectsOnlyKeepsDestinationNote(t *testing.T) {
	m := NewTracker(2, 4, 80, 24)

	arp := audio.ArpeggioEffect{Offsets: []int{0, 4, 7}}
	// Source cell contains note + FX payload.
	m.Tracks[0].Rows[0] = TrackRow{
		Note: notes.Note{Base: notes.BaseE, Octave: 4},
		FX: audio.EffectDefinitions{
			Pitch: audio.PitchEffect{Arpeggio: &arp},
		},
	}

	// Copy source cell.
	m.nav.SetCursorPosition(0, 0)
	m.nav.ClearSelection()
	m.handleGlobalEditingKey("alt+c")

	// Destination has a different note that should be preserved.
	m.Tracks[1].Rows[1] = TrackRow{Note: notes.Note{Base: notes.BaseA, Octave: 5}}
	m.nav.SetCursorPosition(1, 1)

	edited := m.handleGlobalEditingKey("alt+shift+v")
	require.True(t, edited, "expected alt+shift+v to edit destination cell")

	got := m.Tracks[1].Rows[1]
	assert.Equal(t, notes.BaseA, got.Note.Base, "expected destination note base A to be preserved")
	assert.Equal(t, notes.Octave(5), got.Note.Octave, "expected destination note octave 5 to be preserved")
	require.NotNil(t, got.FX.Pitch.Arpeggio, "expected arp copied")
	assert.True(t, got.FX.Pitch.Arpeggio.IsActive(), "expected arp active after copy")
}

func TestNewBPM_ClampsToValidRange(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{160, 160},       // Normal value
		{20, minBPM},     // Below minimum
		{500, maxBPM},    // Above maximum
		{minBPM, minBPM}, // Exactly minimum
		{maxBPM, maxBPM}, // Exactly maximum
	}

	for _, tt := range tests {
		bpm := NewBPM(tt.input)
		assert.Equal(t, tt.expected, bpm.Value(), "NewBPM(%d).Value()", tt.input)
	}
}

func TestBPM_Duration(t *testing.T) {
	bpm := NewBPM(120)
	duration := bpm.Duration()
	expected := 500 * time.Millisecond // 60000ms / 120 BPM = 500ms per beat
	assert.Equal(t, expected, duration, "BPM(120).Duration()")

	// Test edge case: zero BPM should fall back to DefaultBPM
	zeroBPM := BPM{value: 0}
	duration = zeroBPM.Duration()
	expected = time.Duration(60000/DefaultBPM) * time.Millisecond
	assert.Equal(t, expected, duration, "BPM(0).Duration() (DefaultBPM fallback)")
}

func TestBPM_Adjust(t *testing.T) {
	tests := []struct {
		initial  int
		delta    int
		expected int
	}{
		{120, 10, 130},        // Normal adjustment
		{120, -10, 110},       // Negative adjustment
		{minBPM, -10, minBPM}, // Clamp to minimum
		{maxBPM, 10, maxBPM},  // Clamp to maximum
		{100, 250, maxBPM},    // Large positive adjustment
		{100, -100, minBPM},   // Large negative adjustment
	}

	for _, tt := range tests {
		bpm := NewBPM(tt.initial)
		adjusted := bpm.Adjust(tt.delta)
		assert.Equal(t, tt.expected, adjusted.Value(), "BPM(%d).Adjust(%d).Value()", tt.initial, tt.delta)
	}
}

func pressKey(m *TrackerModel, key string) {
	m.Update(tea.KeyPressMsg{Text: key})
}

func TestVolumeColumnHexEntry(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnVolume

	// First key accumulates into buffer but does not commit.
	pressKey(m, "8")
	assert.Equal(t, "8", m.inputBuf, "expected partial buffer after first hex digit")
	assert.False(t, m.Tracks[0].Rows[0].FX.Volume.Active, "volume must not be active after one digit")

	// Second key commits.
	pressKey(m, "0")
	assert.Equal(t, "", m.inputBuf, "buffer must be cleared after commit")
	assert.True(t, m.Tracks[0].Rows[0].FX.Volume.Active, "volume must be active after two digits")
	want := float64(0x80) / 255.0
	assert.InDelta(t, want, m.Tracks[0].Rows[0].FX.Volume.Level, 1e-9, "unexpected volume level")
}

func TestVolumeColumnNavigationDiscardsBuffer(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnVolume

	pressKey(m, "A")
	require.Equal(t, "A", m.inputBuf, "expected partial buffer")

	// Any navigation key must discard the buffer.
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, "", m.inputBuf, "buffer must be cleared on navigation")
	assert.False(t, m.Tracks[0].Rows[0].FX.Volume.Active, "no volume effect must have been committed")
}

func TestFXColumnVibrato(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnFX

	pressKey(m, "v")
	assert.Equal(t, "V", m.inputBuf)
	pressKey(m, "8")
	assert.Equal(t, "V8", m.inputBuf)
	pressKey(m, "4")
	assert.Equal(t, "", m.inputBuf, "buffer cleared after commit")

	vib := m.Tracks[0].Rows[0].FX.Pitch.Vibrato
	require.NotNil(t, vib, "vibrato must be set")
	assert.InDelta(t, 8.0, vib.Rate, 1e-9, "unexpected rate")
	assert.InDelta(t, 1.0, vib.Depth, 1e-9, "unexpected depth (4/4)")
}

func TestFXColumnPortamento(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnFX

	pressKey(m, "p")
	pressKey(m, "0")
	pressKey(m, "8")

	port := m.Tracks[0].Rows[0].FX.Pitch.Portamento
	require.NotNil(t, port, "portamento must be set")
	assert.Equal(t, 8, port.Ticks, "unexpected ticks")
}

func TestFXColumnVolumeSlide(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnFX

	// S80 → int8(0x80) = -128 → delta = -128/127
	pressKey(m, "s")
	pressKey(m, "8")
	pressKey(m, "0")

	slide := m.Tracks[0].Rows[0].FX.VolumeSlide
	assert.True(t, slide.IsActive(), "volume slide must be active")
	// S80 → int8(-128)/127 ≈ -1.008; clamped by NewEffectDefinitions to -1.0.
	assert.InDelta(t, -1.0, slide.Delta, 1e-9, "unexpected slide delta")
}

func TestFXColumnNoteCut(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnFX

	pressKey(m, "c")
	pressKey(m, "0")
	pressKey(m, "3")

	cut := m.Tracks[0].Rows[0].FX.NoteCut
	assert.Equal(t, 3, cut.Tick, "unexpected cut tick")
}

func TestFXColumnNoteDelay(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnFX

	pressKey(m, "d")
	pressKey(m, "0")
	pressKey(m, "2")

	delay := m.Tracks[0].Rows[0].FX.NoteDelay
	assert.Equal(t, 2, delay.Tick, "unexpected delay tick")
}

func TestFXColumnArpPreset(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnFX
	// Give the row some ticks so the arp has offsets to compare.
	m.Tracks[0].Rows[0].FX.Ticks = 4

	// A14 → preset=1 (up), step=4
	pressKey(m, "a")
	pressKey(m, "1")
	pressKey(m, "4")

	arp := m.Tracks[0].Rows[0].FX.Pitch.Arpeggio
	require.NotNil(t, arp, "arpeggio must be set")
	assert.True(t, arp.IsActive(), "arpeggio must be active")
}

func TestFXColumnArpClear(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	arp := audio.ArpeggioEffect{Offsets: []int{0, 4, 7}}
	m.Tracks[0].Rows[0].FX.Pitch = audio.PitchEffect{Arpeggio: &arp}
	m.CursorCol = columnFX

	// A00 → preset nibble 0 → clear arp
	pressKey(m, "a")
	pressKey(m, "0")
	pressKey(m, "0")

	assert.Nil(t, m.Tracks[0].Rows[0].FX.Pitch.Arpeggio, "arpeggio must be cleared by A00")
}

func TestFXColumnPartialInputShownInView(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnFX

	pressKey(m, "s")
	view := m.View()
	assert.Contains(t, view, "S__", "partial buffer S__ must appear in view")

	pressKey(m, "4")
	view = m.View()
	assert.Contains(t, view, "S4_", "partial buffer S4_ must appear in view")
}

func TestFXColumnTabDiscardsBuffer(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnFX

	pressKey(m, "s")
	require.Equal(t, "S", m.inputBuf)

	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	assert.Equal(t, "", m.inputBuf, "tab must discard partial buffer")
}

func TestFXColumnDeleteClearsEffectAndBuffer(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnFX
	vib := audio.VibratoEffect{Rate: 4, Depth: 1}
	m.Tracks[0].Rows[0].FX.Pitch = audio.PitchEffect{Vibrato: &vib}

	// Partial buffer first.
	pressKey(m, "p")
	require.Equal(t, "P", m.inputBuf)

	m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	assert.Equal(t, "", m.inputBuf, "delete must clear buffer")
	assert.Nil(t, m.Tracks[0].Rows[0].FX.Pitch.Vibrato, "vibrato must be cleared by delete")
}
