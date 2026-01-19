package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestFileDialogVisibility(t *testing.T) {
	dialog := NewFileDialog(lipgloss.NewStyle())

	if dialog.IsVisible() {
		t.Error("Dialog should be hidden initially")
	}

	dialog.Show(ModeSave, "test.yaml")
	if !dialog.IsVisible() {
		t.Error("Dialog should be visible after Show()")
	}

	dialog.Hide()
	if dialog.IsVisible() {
		t.Error("Dialog should be hidden after Hide()")
	}
}

func TestFileDialogPrefill(t *testing.T) {
	dialog := NewFileDialog(lipgloss.NewStyle())

	dialog.Show(ModeSave, "mysong.yaml")
	if dialog.Input != "mysong.yaml" {
		t.Errorf("Expected Input='mysong.yaml', got '%s'", dialog.Input)
	}
	if dialog.Mode != ModeSave {
		t.Errorf("Expected Mode=ModeSave, got %v", dialog.Mode)
	}
}

func TestFileDialogInput(t *testing.T) {
	dialog := NewFileDialog(lipgloss.NewStyle())
	dialog.Show(ModeSave, "")

	// Type some characters
	*dialog, _ = dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	*dialog, _ = dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	*dialog, _ = dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	*dialog, _ = dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	if dialog.Input != "test" {
		t.Errorf("Expected Input='test', got '%s'", dialog.Input)
	}

	// Test backspace
	*dialog, _ = dialog.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if dialog.Input != "tes" {
		t.Errorf("Expected Input='tes' after backspace, got '%s'", dialog.Input)
	}
}

func TestFileDialogConfirm(t *testing.T) {
	dialog := NewFileDialog(lipgloss.NewStyle())
	dialog.Show(ModeSave, "test")

	// Press enter
	_, cmd := dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Expected command to be returned")
	}

	msg := cmd()
	confirmed, ok := msg.(FileDialogConfirmed)
	if !ok {
		t.Fatal("Expected FileDialogConfirmed message")
	}

	if confirmed.Filename != "test.yaml" {
		t.Errorf("Expected Filename='test.yaml', got '%s'", confirmed.Filename)
	}
}

func TestFileDialogCancel(t *testing.T) {
	dialog := NewFileDialog(lipgloss.NewStyle())
	dialog.Show(ModeSave, "test")

	// Press escape
	_, cmd := dialog.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("Expected command to be returned")
	}

	msg := cmd()
	_, ok := msg.(FileDialogCancelled)
	if !ok {
		t.Fatal("Expected FileDialogCancelled message")
	}
}

func TestFileDialogEmptyFilename(t *testing.T) {
	dialog := NewFileDialog(lipgloss.NewStyle())
	dialog.Show(ModeSave, "")

	// Try to confirm with empty filename
	*dialog, _ = dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if dialog.Error == "" {
		t.Error("Expected error for empty filename")
	}
}
