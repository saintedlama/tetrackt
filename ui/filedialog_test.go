package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFileDialogModeSet(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "")
	if dialog.Mode != ModeSave {
		t.Errorf("Expected Mode=ModeSave, got %v", dialog.Mode)
	}
}

func TestFileDialogPrefill(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "mysong.json")
	if dialog.InputValue() != "mysong.json" {
		t.Errorf("Expected InputValue='mysong.json', got '%s'", dialog.InputValue())
	}
	if dialog.Mode != ModeSave {
		t.Errorf("Expected Mode=ModeSave, got %v", dialog.Mode)
	}
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

	if dialog.InputValue() != "test" {
		t.Errorf("Expected InputValue='test', got '%s'", dialog.InputValue())
	}

	model, _ = dialog.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	dialog = model.(*FileDialogModel)
	if dialog.InputValue() != "tes" {
		t.Errorf("Expected InputValue='tes' after backspace, got '%s'", dialog.InputValue())
	}
}

func TestFileDialogConfirm(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "test")
	dialog.FocusInput()

	_, cmd := dialog.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Expected command to be returned")
	}

	msg := cmd()
	closeMsg, ok := msg.(CloseDialogMsg)
	if !ok {
		t.Fatal("Expected CloseDialogMsg")
	}

	confirmed, ok := closeMsg.Payload.(FileDialogConfirmed)
	if !ok {
		t.Fatal("Expected FileDialogConfirmed payload")
	}

	if !strings.HasSuffix(confirmed.Filename, "test.json") {
		t.Errorf("Expected Filename to end with 'test.json', got '%s'", confirmed.Filename)
	}
	if confirmed.Mode != ModeSave {
		t.Errorf("Expected Mode=ModeSave, got %v", confirmed.Mode)
	}
}

func TestFileDialogCancel(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "test")

	_, cmd := dialog.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Expected command to be returned")
	}

	msg := cmd()
	closeMsg, ok := msg.(CloseDialogMsg)
	if !ok {
		t.Fatal("Expected CloseDialogMsg")
	}

	if closeMsg.Payload != nil {
		t.Error("Expected nil Payload for cancel")
	}
}

func TestFileDialogEmptyFilename(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "")
	dialog.FocusInput()

	_, cmd := dialog.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Error("Expected nil command for empty filename")
	}
	if dialog.ErrMsg == "" {
		t.Error("Expected error message for empty filename")
	}
}
