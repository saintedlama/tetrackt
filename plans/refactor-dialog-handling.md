# Refactor Modal Handling

The dialog handlingis currently implemented per instance, leading to code duplication and inconsistent behavior across different dialogs. To improve maintainability and user experience, we should refactor the dialog handling into a more centralized and reusable system.

pcloud-cli implements a generic dialog component here: <https://github.com/saintedlama/pcloud-cli/blob/master/internal/tui/dialog.go> and an overlay component to render dialogs on top of the main UI here: <https://github.com/saintedlama/pcloud-cli/blob/master/internal/tui/overlay.go>. We can take inspiration from this implementation to create a similar system for TeTrackT, allowing us to manage dialogs more effectively and provide a consistent user experience across the application.
