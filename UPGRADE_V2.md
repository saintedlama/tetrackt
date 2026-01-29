# Upgrade to Bubbletea v2 and Lipgloss v2 - Status Report

## Summary

This document tracks the upgrade attempt from bubbletea v1 and lipgloss v1 to their v2 versions.

## Current Status: ⚠️ BLOCKED

The upgrade to v2 cannot be completed at this time due to compatibility issues between pre-release versions.

## Attempted Versions

- **Bubbletea**: v2.0.0-rc.2 (Release Candidate 2)
- **Lipgloss**: v2.0.0-beta.3 (Beta 3)

## Issues Encountered

### 1. Module Path Inconsistency

Bubbletea v2 declares its module path as `charm.land/bubbletea/v2`, but Lipgloss v2 uses `github.com/charmbracelet/lipgloss/v2`. This creates a dependency resolution conflict.

### 2. Dependency Version Mismatches

The pre-release versions have incompatible dependencies:

- `github.com/charmbracelet/x/ansi` API changes between v0.10.1 and v0.11.x
- `github.com/charmbracelet/x/cellbuf` incompatibilities between v0.0.13 and v0.0.14  
- Method signature changes in ansi.Style (methods now require boolean arguments)

### 3. Build Errors

When attempting to build with v2 dependencies, the following errors occur:

```
# github.com/charmbracelet/lipgloss/v2
C:\Users\chris\go\pkg\mod\github.com\charmbracelet\lipgloss\v2@v2.0.0-beta.3\style.go:297:8: not enough arguments in call to te.Italic
C:\Users\chris\go\pkg\mod\github.com\charmbracelet\lipgloss\v2@v2.0.0-beta.3\style.go:300:8: not enough arguments in call to te.Underline
C:\Users\chris\go\pkg\mod\github.com\charmbracelet\lipgloss\v2@v2.0.0-beta.3\style.go:307:11: te.SlowBlink undefined
...
```

### 4. Pre-Release Instability

As of January 2026, v2.0.0 stable has not been released. The current versions are:
- Bubbletea: v2.0.0-rc.2 (Release Candidate)
- Lipgloss: v2.0.0-beta.3 (Beta)

These pre-release versions were not tested together and have incompatible dependency requirements.

## What's New in V2

### Bubbletea v2

According to the official documentation:

1. **Cursed Renderer**: Ground-up rewritten renderer based on ncurses algorithms for better speed and accuracy
2. **Enhanced Keyboard Handling**: Support for modifier keys (Shift+Enter) and key releases
3. **Improved I/O Management**: Bubble Tea now fully manages I/O and orchestrates with companion libraries
4. **Built-in Color Downsampling**: Automatic color profile detection and ANSI downsampling

### Lipgloss v2

1. **Deterministic Styles**: More predictable style behavior  
2. **Intentional I/O**: Precise control over input/output devices
3. **Compositing Layers**: New canvas APIs for grouping and arranging styled outputs
4. **Table Enhancements**: Improved table rendering and APIs
5. **Better Bubble Tea v2 Integration**: Designed to work seamlessly with Bubble Tea v2

## Recommended Actions

### Option 1: Wait for Stable Release (RECOMMENDED)

Wait for the official stable v2.0.0 release of both libraries. The Charm team is actively working on v2, and a stable release should resolve these compatibility issues.

**Timeline**: Unknown, but RC2 suggests release is near

### Option 2: Stay on V1

Continue using the stable v1 versions until v2 is production-ready:
- `github.com/charmbracelet/bubbletea` v1.3.10
- `github.com/charmbracelet/lipgloss` v1.1.0

These versions are stable, well-tested, and fully functional.

### Option 3: Track V2 Development

Create a separate development branch to periodically test v2 compatibility as new pre-release versions are published.

## Migration Checklist (for when V2 stabilizes)

- [ ] Update import paths in all files
  - [ ] main.go
  - [ ] ui/tracker.go
  - [ ] ui/envelope.go
  - [ ] ui/filedialog.go
  - [ ] ui/filedialog_test.go
  - [ ] ui/oscilator.go
  - [ ] ui/presets.go
  - [ ] ui/onoff.go
  - [ ] ui/bar.go
  - [ ] ui/mixer.go

- [ ] Update go.mod
  - [ ] Determine final module paths (charm.land vs github.com)
  - [ ] Update to stable v2.0.0 versions
  - [ ] Run `go mod tidy`

- [ ] Test for API changes
  - [ ] Review migration guides
  - [ ] Update any changed APIs
  - [ ] Test all functionality

- [ ] Run tests
  - [ ] `go test ./...`
  - [ ] Manual testing of UI

## Resources

- [Bubble Tea v2 Discussion](https://github.com/charmbracelet/bubbletea/discussions/1374)
- [Lipgloss v2 Migration Guide](https://github.com/charmbracelet/lipgloss/discussions/506)
- [Bubble Tea Releases](https://github.com/charmbracelet/bubbletea/releases)
- [Lipgloss Releases](https://github.com/charmbracelet/lipgloss/releases)

## Conclusion

The v2 upgrade should be attempted again once stable releases are available. In the meantime, the current v1 versions provide all necessary functionality and should remain in use.

---

**Last Updated**: January 28, 2026
**Status**: Waiting for stable v2 release
**Next Review**: Check for v2.0.0 stable release monthly
