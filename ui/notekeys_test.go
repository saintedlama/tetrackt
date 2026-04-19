package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetrackt/tetrackt/notes"
)

func TestInputProfileDefaultsToQWERTY(t *testing.T) {
	SetInputProfile(InputProfileQWERTY)
	require.Equal(t, InputProfileQWERTY, CurrentInputProfile(), "expected qwerty profile")
	assert.Equal(t, notes.Base("C"), NoteKeys["z"], "expected z->C in qwerty")
	assert.Equal(t, notes.Base("A"), NoteKeys["y"], "expected y->A in qwerty")
}

func TestInputProfileQWERTZSwapsYZRows(t *testing.T) {
	SetInputProfile(InputProfileQWERTZ)
	require.Equal(t, InputProfileQWERTZ, CurrentInputProfile(), "expected qwertz profile")
	assert.Equal(t, notes.Base("C"), NoteKeys["y"], "expected y->C in qwertz")
	assert.Equal(t, notes.Base("A"), NoteKeys["z"], "expected z->A in qwertz")
}

func TestInputProfileFromStringFallback(t *testing.T) {
	assert.Equal(t, InputProfileQWERTZ, InputProfileFromString("qwertz"), "expected qwertz parse")
	assert.Equal(t, InputProfileQWERTY, InputProfileFromString("unknown"), "expected fallback qwerty parse")
}
