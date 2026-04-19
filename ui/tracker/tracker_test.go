package tracker

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetrackt/tetrackt/audio"
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
	assert.Equal(t, audio.Base("C"), noteMsg.Note.Base, "unexpected note base")
	assert.Equal(t, audio.Octave(4), noteMsg.Note.Octave, "unexpected note octave")
	assert.Equal(t, audio.Base("C"), m.Tracks[0].Rows[0].Note.Base, "expected C note in current cell")
}

func TestEditStepAdvancesCursor(t *testing.T) {
	m := NewTracker(2, 16, 80, 24)
	m.Octave = 4
	m.EditStep = 2
	m.CursorCol = columnNote

	_, _ = m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	assert.Equal(t, 2, m.nav.CursorRow(), "expected cursor row 2")
}

func TestTabMovesToNextSubcolumn(t *testing.T) {
	m := NewTracker(2, 8, 80, 24)
	require.Equal(t, columnNote, m.CursorCol, "expected initial note column")

	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	assert.Equal(t, columnVolume, m.CursorCol, "expected volume column after tab")
}

func TestInsertTrackSpaceShiftsRowsDown(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.nav.SetCursorPosition(0, 1)
	m.Tracks[0].Rows[1].Note = audio.Note{Base: audio.Base("C"), Octave: 4}
	m.Tracks[0].Rows[2].Note = audio.Note{Base: audio.Base("D"), Octave: 4}

	edited := m.handleGlobalEditingKey("insert")
	require.True(t, edited, "expected insert to be treated as edit")

	assert.Equal(t, audio.BaseOff, m.Tracks[0].Rows[1].Note.Base, "expected inserted empty row at cursor")
	assert.Equal(t, audio.Base("C"), m.Tracks[0].Rows[2].Note.Base, "expected previous row shifted down to row 2")
}

func TestTransposeCurrentStoredNoteSemitone(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.nav.SetCursorPosition(0, 0)
	m.Tracks[0].Rows[0].Note = audio.Note{Base: audio.BaseC, Octave: 4}

	edited := m.handleGlobalEditingKey("alt+up")
	require.True(t, edited, "expected transpose to edit current note")

	got := m.Tracks[0].Rows[0].Note
	assert.Equal(t, audio.BaseCs, got.Base, "expected C# after transpose")
	assert.Equal(t, audio.Octave(4), got.Octave, "expected octave 4 after transpose")
}

func TestTransposeSelectedNotesOctave(t *testing.T) {
	m := NewTracker(2, 4, 80, 24)
	m.Tracks[0].Rows[0].Note = audio.Note{Base: audio.BaseC, Octave: 4}
	m.Tracks[1].Rows[1].Note = audio.Note{Base: audio.BaseE, Octave: 4}

	// Use navigation grid to set up selection
	m.nav.SetCursorPosition(0, 0)
	m.nav.MoveExtending(1, 1)

	edited := m.handleGlobalEditingKey("alt+shift+up")
	require.True(t, edited, "expected transpose to edit selected notes")

	n0 := m.Tracks[0].Rows[0].Note
	assert.Equal(t, audio.BaseC, n0.Base, "expected C")
	assert.Equal(t, audio.Octave(5), n0.Octave, "expected C5")

	n1 := m.Tracks[1].Rows[1].Note
	assert.Equal(t, audio.BaseE, n1.Base, "expected E")
	assert.Equal(t, audio.Octave(5), n1.Octave, "expected E5")
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
		"o": EffectContinuous,
		"a": EffectArpPreset,
	}

	for key, want := range cases {
		got, ok := effects.ParseKey(key)
		require.True(t, ok, "key %q: expected ok=true", key)
		assert.Equal(t, want, EffectType(got), "key %q expected %v", key, want)
	}
}

func TestInlineRowTicksCommand(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnEffect

	_, _ = m.Update(tea.KeyPressMsg{Text: "5"})
	assert.Equal(t, EffectRowTicks, m.Tracks[0].Rows[0].Effect.Type, "expected EffectRowTicks")

	m.CursorCol = columnParam
	_, _ = m.Update(tea.KeyPressMsg{Text: "0"})
	_, _ = m.Update(tea.KeyPressMsg{Text: "C"})

	row := m.Tracks[0].Rows[0]
	assert.Equal(t, 12, row.Ticks, "expected ticks=12")
	assert.Equal(t, 0x0C, row.Effect.Param, "expected effect param 0x0C")
}

func TestInlineContinuousCommandToggle(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnEffect

	_, _ = m.Update(tea.KeyPressMsg{Text: "6"})
	m.CursorCol = columnParam
	_, _ = m.Update(tea.KeyPressMsg{Text: "0"})
	_, _ = m.Update(tea.KeyPressMsg{Text: "1"})

	assert.True(t, m.Tracks[0].Rows[0].Continuous, "expected continuous to be enabled")

	m.nav.Move(0, 1)
	m.CursorCol = columnEffect
	_, _ = m.Update(tea.KeyPressMsg{Text: "6"})
	m.CursorCol = columnParam
	_, _ = m.Update(tea.KeyPressMsg{Text: "0"})
	_, _ = m.Update(tea.KeyPressMsg{Text: "0"})

	assert.False(t, m.Tracks[0].Rows[1].Continuous, "expected continuous to be disabled")
}

func TestInlineArpPresetCommand(t *testing.T) {
	m := NewTracker(1, 8, 80, 24)
	m.CursorCol = columnEffect
	_, _ = m.Update(tea.KeyPressMsg{Text: "7"})

	m.CursorCol = columnParam
	_, _ = m.Update(tea.KeyPressMsg{Text: "1"})
	_, _ = m.Update(tea.KeyPressMsg{Text: "4"})

	row := m.Tracks[0].Rows[0]
	assert.Equal(t, EffectArpPreset, row.Effect.Type, "expected EffectArpPreset")
	assert.True(t, row.Arpeggio.IsActive(), "expected arp offsets to be generated")
	assert.True(t, row.Continuous, "expected arp preset command to force continuous")
}

func TestTransposeAltShiftOrderVariants(t *testing.T) {
	m := NewTracker(1, 4, 80, 24)
	m.nav.SetCursorPosition(0, 0)
	m.Tracks[0].Rows[0].Note = audio.Note{Base: audio.BaseC, Octave: 4}

	edited := m.handleGlobalEditingKey("shift+alt+up")
	require.True(t, edited, "expected shift+alt+up to transpose")

	got := m.Tracks[0].Rows[0].Note
	assert.Equal(t, audio.BaseC, got.Base, "expected C")
	assert.Equal(t, audio.Octave(5), got.Octave, "expected octave 5 after shift+alt+up")

	edited = m.handleGlobalEditingKey("shift+alt+down")
	require.True(t, edited, "expected shift+alt+down to transpose")

	got = m.Tracks[0].Rows[0].Note
	assert.Equal(t, audio.BaseC, got.Base, "expected C")
	assert.Equal(t, audio.Octave(4), got.Octave, "expected octave 4 after shift+alt+down")
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
	require.True(t, edited, "expected alt+shift+v to edit destination cell")

	got := m.Tracks[1].Rows[1]
	assert.Equal(t, audio.BaseA, got.Note.Base, "expected destination note base A to be preserved")
	assert.Equal(t, audio.Octave(5), got.Note.Octave, "expected destination note octave 5 to be preserved")
	assert.Equal(t, 48, got.Volume, "expected volume copied")
	assert.Equal(t, 12, got.Ticks, "expected ticks copied")
	assert.True(t, got.Continuous, "expected continuous copied")
	assert.True(t, got.Arpeggio.IsActive(), "expected arp copied")
	assert.Equal(t, EffectVibrato, got.Effect.Type, "expected effect type copied")
	assert.Equal(t, 0x24, got.Effect.Param, "expected effect param copied")
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
