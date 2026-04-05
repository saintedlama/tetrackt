package ui

import (
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
	dialog := NewFileDialog(ModeSave, "mysong.yaml")
	if dialog.Input != "mysong.yaml" {
		t.Errorf("Expected Input='mysong.yaml', got '%s'", dialog.Input)
	}
	if dialog.Mode != ModeSave {
		t.Errorf("Expected Mode=ModeSave, got %v", dialog.Mode)
	}
}

func TestFileDialogInput(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "")

	// Type some characters
	model, _ := dialog.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	dialog = model.(*FileDialogModel)
	model, _ = dialog.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	dialog = model.(*FileDialogModel)
	model, _ = dialog.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	dialog = model.(*FileDialogModel)
	model, _ = dialog.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	dialog = model.(*FileDialogModel)

	if dialog.Input != "test" {
		t.Errorf("Expected Input='test', got '%s'", dialog.Input)
	}

	// Test backspace
	model, _ = dialog.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	dialog = model.(*FileDialogModel)
	if dialog.Input != "tes" {
		t.Errorf("Expected Input='tes' after backspace, got '%s'", dialog.Input)
	}
}

func TestFileDialogConfirm(t *testing.T) {
	dialog := NewFileDialog(ModeSave, "test")

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

	if confirmed.Filename != "test.yaml" {
		t.Errorf("Expected Filename='test.yaml', got '%s'", confirmed.Filename)
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

	model, _ := dialog.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	dialog = model.(*FileDialogModel)

	if dialog.Error == "" {
		t.Error("Expected error for empty filename")
	}
}
