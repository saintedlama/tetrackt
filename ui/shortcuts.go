package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// ShortcutAction executes a keyboard shortcut handler.
// It returns whether the key was handled and an optional command.
type ShortcutAction func(tea.KeyPressMsg) (bool, tea.Cmd)

// Shortcut defines a key binding, its help description, and optional behavior.
type Shortcut struct {
	Keys        []string
	KeyLabel    string
	Description string
	Action      ShortcutAction
	Matcher     func(tea.KeyPressMsg) bool
	Hidden      bool
}

// ShortcutSection groups related shortcuts for dispatching and help rendering.
type ShortcutSection struct {
	Title     string
	Shortcuts []Shortcut
}

// DispatchShortcutSections executes the first matching shortcut action.
func DispatchShortcutSections(msg tea.KeyPressMsg, sections []ShortcutSection) (bool, tea.Cmd) {
	for _, section := range sections {
		for _, shortcut := range section.Shortcuts {
			if shortcut.Action == nil {
				continue
			}
			if !shortcutMatches(shortcut, msg) {
				continue
			}
			handled, cmd := shortcut.Action(msg)
			if handled {
				return true, cmd
			}
		}
	}

	return false, nil
}

// HelpSectionsFromShortcutSections converts shortcut definitions to help sections.
func HelpSectionsFromShortcutSections(sections []ShortcutSection) []HelpSection {
	out := make([]HelpSection, 0, len(sections))

	for _, section := range sections {
		entries := make([]HelpEntry, 0, len(section.Shortcuts))
		for _, shortcut := range section.Shortcuts {
			if shortcut.Hidden {
				continue
			}
			if shortcut.KeyLabel == "" && len(shortcut.Keys) == 0 && shortcut.Description == "" {
				entries = append(entries, HelpEntry{Key: "", Desc: ""})
				continue
			}
			entries = append(entries, HelpEntry{
				Key:  helpKeyLabel(shortcut),
				Desc: shortcut.Description,
			})
		}

		if len(entries) == 0 {
			continue
		}

		out = append(out, HelpSection{Title: section.Title, Entries: entries})
	}

	return out
}

func shortcutMatches(shortcut Shortcut, msg tea.KeyPressMsg) bool {
	if shortcut.Matcher != nil {
		return shortcut.Matcher(msg)
	}

	key := msg.String()
	for _, candidate := range shortcut.Keys {
		if key == candidate {
			return true
		}
	}

	return false
}

func helpKeyLabel(shortcut Shortcut) string {
	if shortcut.KeyLabel != "" {
		return shortcut.KeyLabel
	}
	return strings.Join(shortcut.Keys, " / ")
}
