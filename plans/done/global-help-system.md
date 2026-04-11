# Global Keyboard Shortcut Help System

**Status:** Done

Currently screens and main implement part of the help system by rendering a static help text panel. This is inflexible and hard to maintain. Instead, implement a global help system that can be invoked from any screen with `?` and supports multiple pages of content.

For dialogs, the help system is left encapsulated within the dialog, as the help content is specific to the dialog's context.

## How it works

- Pressing `?` on any screen opens the help overlay.
- The help system shows global shortcuts.
- The help system shows screen-specific shortcuts for the current screen.
