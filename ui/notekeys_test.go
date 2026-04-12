package ui

import "testing"

func TestInputProfileDefaultsToQWERTY(t *testing.T) {
	SetInputProfile(InputProfileQWERTY)
	if CurrentInputProfile() != InputProfileQWERTY {
		t.Fatalf("expected qwerty profile, got %s", CurrentInputProfile())
	}
	if got := NoteKeys["z"]; got != "C" {
		t.Fatalf("expected z->C in qwerty, got %q", got)
	}
	if got := NoteKeys["y"]; got != "A" {
		t.Fatalf("expected y->A in qwerty, got %q", got)
	}
}

func TestInputProfileQWERTZSwapsYZRows(t *testing.T) {
	SetInputProfile(InputProfileQWERTZ)
	if CurrentInputProfile() != InputProfileQWERTZ {
		t.Fatalf("expected qwertz profile, got %s", CurrentInputProfile())
	}
	if got := NoteKeys["y"]; got != "C" {
		t.Fatalf("expected y->C in qwertz, got %q", got)
	}
	if got := NoteKeys["z"]; got != "A" {
		t.Fatalf("expected z->A in qwertz, got %q", got)
	}
}

func TestInputProfileFromStringFallback(t *testing.T) {
	if got := InputProfileFromString("qwertz"); got != InputProfileQWERTZ {
		t.Fatalf("expected qwertz parse, got %s", got)
	}
	if got := InputProfileFromString("unknown"); got != InputProfileQWERTY {
		t.Fatalf("expected fallback qwerty parse, got %s", got)
	}
}
