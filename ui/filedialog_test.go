package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileDialogModeSet(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "")
	assert.Equal(t, ModeSave, dialog.Mode, "expected Mode=ModeSave")
}

func TestFileDialogPrefill(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "mymodule.json")
	assert.Equal(t, "mymodule.json", dialog.InputValue(), "expected InputValue='mymodule.json'")
	assert.Equal(t, ModeSave, dialog.Mode, "expected Mode=ModeSave")
}

func TestFileDialogInput(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "")
	dialog.FocusInput()

	model, _ := dialog.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	dialog = model.(*FileDialogModel)
	model, _ = dialog.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	dialog = model.(*FileDialogModel)
	model, _ = dialog.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	dialog = model.(*FileDialogModel)
	model, _ = dialog.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	dialog = model.(*FileDialogModel)

	assert.Equal(t, "test", dialog.InputValue(), "expected InputValue='test'")

	model, _ = dialog.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	dialog = model.(*FileDialogModel)
	assert.Equal(t, "tes", dialog.InputValue(), "expected InputValue='tes' after backspace")
}

func TestFileDialogConfirm(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "test")
	dialog.FocusInput()

	_, cmd := dialog.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd, "expected command to be returned")

	msg := cmd()
	closeMsg, ok := msg.(CloseDialogMsg)
	require.True(t, ok, "expected CloseDialogMsg")

	confirmed, ok := closeMsg.Payload.(FileDialogConfirmed)
	require.True(t, ok, "expected FileDialogConfirmed payload")

	assert.True(t, strings.HasSuffix(confirmed.Filename, "test.json"), "expected Filename to end with 'test.json', got %q", confirmed.Filename)
	assert.Equal(t, ModeSave, confirmed.Mode, "expected Mode=ModeSave")
}

func TestFileDialogCancel(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "test")

	_, cmd := dialog.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	require.NotNil(t, cmd, "expected command to be returned")

	msg := cmd()
	closeMsg, ok := msg.(CloseDialogMsg)
	require.True(t, ok, "expected CloseDialogMsg")

	assert.Nil(t, closeMsg.Payload, "expected nil Payload for cancel")
}

func TestFileDialogEmptyFilename(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "")
	dialog.FocusInput()

	_, cmd := dialog.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd, "expected nil command for empty filename")
	assert.NotEmpty(t, dialog.ErrMsg, "expected error message for empty filename")
}
