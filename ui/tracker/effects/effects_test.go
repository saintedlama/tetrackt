package effects

import (
	"testing"
)

func TestParseNibble_SupportsInlineExtensions(t *testing.T) {
	for i := 0; i <= int(ArpPreset); i++ {
		typ, ok := ParseNibble(i)
		if !ok {
			t.Fatalf("expected nibble %d to be valid", i)
		}
		if int(typ) != i {
			t.Fatalf("expected type %d, got %d", i, typ)
		}
	}

	if _, ok := ParseNibble(8); ok {
		t.Fatal("expected nibble 8 to be invalid")
	}
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
		if !ok || got != want {
			t.Fatalf("key %q expected %v, got %v (ok=%v)", key, want, got, ok)
		}
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
		got := typ.Format()
		if got != want {
			t.Fatalf("type %d expected format %q, got %q", typ, want, got)
		}
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
		got := tt.typ.FormatParam(tt.param)
		if got != tt.want {
			t.Fatalf("type %d param %d: expected %q, got %q", tt.typ, tt.param, tt.want, got)
		}
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
		if gotOk != tt.wantOk || (gotOk && gotVal != tt.wantVal) {
			t.Fatalf("type %d param %d: expected (%d, %v), got (%d, %v)",
				tt.typ, tt.param, tt.wantVal, tt.wantOk, gotVal, gotOk)
		}
	}
}

func TestApply_RowTicks(t *testing.T) {
	result, ok := Apply(RowTicks, 0x0C, 0, 6)
	if !ok {
		t.Fatal("expected apply to succeed")
	}
	if result.Ticks != 12 {
		t.Fatalf("expected ticks=12, got %d", result.Ticks)
	}
	if result.Effect.Type != RowTicks {
		t.Fatalf("expected effect type RowTicks, got %v", result.Effect.Type)
	}
}

func TestApply_Continuous(t *testing.T) {
	result, ok := Apply(Continuous, 1, 0, 6)
	if !ok {
		t.Fatal("expected apply to succeed")
	}
	if !result.Continuous {
		t.Fatal("expected continuous to be enabled")
	}

	result, ok = Apply(Continuous, 0, 0, 6)
	if !ok {
		t.Fatal("expected apply to succeed")
	}
	if result.Continuous {
		t.Fatal("expected continuous to be disabled")
	}
}

func TestApply_ArpPreset(t *testing.T) {
	// Preset 1, step 4 (default)
	result, ok := Apply(ArpPreset, 0x14, 0, 6)
	if !ok {
		t.Fatal("expected apply to succeed")
	}
	if !result.Arpeggio.IsActive() {
		t.Fatal("expected arp offsets to be generated")
	}
	if !result.Continuous {
		t.Fatal("expected arp preset command to force continuous")
	}
	// Should generate 6 offsets (defaultSpeed)
	if len(result.Arpeggio.Offsets) != 6 {
		t.Fatalf("expected 6 offsets, got %d", len(result.Arpeggio.Offsets))
	}
}

func TestApply_ArpPreset_ZeroClearsArp(t *testing.T) {
	result, ok := Apply(ArpPreset, 0x00, 12, 6)
	if !ok {
		t.Fatal("expected apply to succeed")
	}
	if result.Arpeggio.IsActive() {
		t.Fatal("expected arpeggio to be cleared")
	}
}

func TestGenerateArpOffsets_Up(t *testing.T) {
	offsets := generateArpOffsets(1, 4, 3)
	want := []int{0, 3, 6, 9}
	if len(offsets) != len(want) {
		t.Fatalf("expected %d offsets, got %d", len(want), len(offsets))
	}
	for i, v := range want {
		if offsets[i] != v {
			t.Fatalf("offset[%d]: expected %d, got %d", i, v, offsets[i])
		}
	}
}

func TestGenerateArpOffsets_Down(t *testing.T) {
	offsets := generateArpOffsets(2, 4, 3)
	want := []int{9, 6, 3, 0}
	if len(offsets) != len(want) {
		t.Fatalf("expected %d offsets, got %d", len(want), len(offsets))
	}
	for i, v := range want {
		if offsets[i] != v {
			t.Fatalf("offset[%d]: expected %d, got %d", i, v, offsets[i])
		}
	}
}
