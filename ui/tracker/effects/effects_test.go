package effects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNibble_SupportsInlineExtensions(t *testing.T) {
	for i := 0; i <= int(ArpPreset); i++ {
		typ, ok := ParseNibble(i)
		require.True(t, ok, "expected nibble %d to be valid", i)
		assert.Equal(t, i, int(typ), "expected type %d", i)
	}

	_, ok := ParseNibble(8)
	assert.False(t, ok, "expected nibble 8 to be invalid")
}

func TestParseKey_Aliases(t *testing.T) {
	cases := map[string]Type{
		"v": Vibrato,
		"s": VolumeSlide,
		"c": NoteCut,
		"d": NoteDelay,
		"t": RowTicks,
		"o": Continuous,
		"a": ArpPreset,
	}

	for key, want := range cases {
		got, ok := ParseKey(key)
		require.True(t, ok, "key %q: expected ok=true", key)
		assert.Equal(t, want, got, "key %q expected %v", key, want)
	}
}

func TestType_Format(t *testing.T) {
	cases := map[Type]string{
		None:        ".",
		Vibrato:     "V",
		VolumeSlide: "S",
		NoteCut:     "C",
		NoteDelay:   "D",
		RowTicks:    "T",
		Continuous:  "O",
		ArpPreset:   "A",
	}

	for typ, want := range cases {
		assert.Equal(t, want, typ.Format(), "type %d", typ)
	}
}

func TestType_FormatParam(t *testing.T) {
	tests := []struct {
		typ   Type
		param int
		want  string
	}{
		{Vibrato, 0x24, "24"},
		{Vibrato, 255, "FF"},
		{Vibrato, 0, "00"},
		{VolumeSlide, -8, "F8"}, // negative signed byte
		{VolumeSlide, 8, "08"},
		{RowTicks, 12, "0C"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.typ.FormatParam(tt.param), "type %d param %d", tt.typ, tt.param)
	}
}

func TestType_DecodeParam(t *testing.T) {
	tests := []struct {
		typ     Type
		param   int
		wantVal int
		wantOk  bool
	}{
		{Vibrato, 0x24, 0x24, true},
		{VolumeSlide, 0xF8, -8, true}, // signed byte
		{RowTicks, 0, 0, true},
		{RowTicks, 12, 12, true},
		{RowTicks, 33, 0, false}, // out of range
		{Continuous, 0, 0, true},
		{Continuous, 1, 1, true},
		{Continuous, 2, 0, false}, // invalid
	}

	for _, tt := range tests {
		gotVal, gotOk := tt.typ.DecodeParam(tt.param)
		assert.Equal(t, tt.wantOk, gotOk, "type %d param %d: ok", tt.typ, tt.param)
		if tt.wantOk {
			assert.Equal(t, tt.wantVal, gotVal, "type %d param %d: value", tt.typ, tt.param)
		}
	}
}

func TestApply_RowTicks(t *testing.T) {
	result, ok := Apply(RowTicks, 0x0C, 0, 6)
	require.True(t, ok, "expected apply to succeed")
	assert.Equal(t, 12, result.Speed, "expected speed=12")
	assert.Equal(t, RowTicks, result.Effect.Type, "expected effect type RowTicks")
}

func TestApply_Continuous(t *testing.T) {
	result, ok := Apply(Continuous, 1, 0, 6)
	require.True(t, ok, "expected apply to succeed")
	assert.True(t, result.Continuous, "expected continuous to be enabled")

	result, ok = Apply(Continuous, 0, 0, 6)
	require.True(t, ok, "expected apply to succeed")
	assert.False(t, result.Continuous, "expected continuous to be disabled")
}

func TestApply_ArpPreset(t *testing.T) {
	// Preset 1, step 4 (default)
	result, ok := Apply(ArpPreset, 0x14, 0, 6)
	require.True(t, ok, "expected apply to succeed")
	assert.True(t, result.Arpeggio.IsActive(), "expected arp offsets to be generated")
	assert.True(t, result.Continuous, "expected arp preset command to force continuous")
	// Should generate 6 offsets (defaultSpeed)
	assert.Len(t, result.Arpeggio.Offsets, 6, "expected 6 offsets")
}

func TestApply_ArpPreset_ZeroClearsArp(t *testing.T) {
	result, ok := Apply(ArpPreset, 0x00, 12, 6)
	require.True(t, ok, "expected apply to succeed")
	assert.False(t, result.Arpeggio.IsActive(), "expected arpeggio to be cleared")
}

func TestGenerateArpOffsets_Up(t *testing.T) {
	offsets := generateArpOffsets(1, 4, 3)
	want := []int{0, 3, 6, 9}
	require.Len(t, offsets, len(want), "expected %d offsets", len(want))
	for i, v := range want {
		assert.Equal(t, v, offsets[i], "offset[%d]", i)
	}
}

func TestGenerateArpOffsets_Down(t *testing.T) {
	offsets := generateArpOffsets(2, 4, 3)
	want := []int{9, 6, 3, 0}
	require.Len(t, offsets, len(want), "expected %d offsets", len(want))
	for i, v := range want {
		assert.Equal(t, v, offsets[i], "offset[%d]", i)
	}
}
