package main_test

import (
	"context"
	"slices"
	"testing"

	"github.com/saintedlama/archscout"
	"github.com/stretchr/testify/require"
)

const module archscout.Module = "github.com/tetrackt/tetrackt"

func loadWorkspace(t *testing.T) *archscout.Workspace {
	t.Helper()
	ws, err := archscout.LoadWorkspace(context.Background(), ".", archscout.WithInMemoryCache())
	require.NoError(t, err, "failed to load workspace")
	return ws
}

// TestArch_AudioIsPure enforces that the synthesis engine has no dependency
// on any UI, persistence, or render package — keeping it a pure audio domain.
func TestArch_AudioIsPure(t *testing.T) {
	ws := loadWorkspace(t)

	archscout.Rule("audio must not depend on ui, persistence, or render").
		Dependencies().
		InPackage(module.Pkg("audio/...")).
		IsNotTest().
		DependOn(module.Pkgs("ui/...", "persistence/...", "render/...")...).
		Test(t, ws)
}

// TestArch_UICommonIsALeaf enforces that ui/common never imports other
// internal packages, keeping it a stable base that anything can depend on.
func TestArch_UICommonIsALeaf(t *testing.T) {
	ws := loadWorkspace(t)

	archscout.Rule("ui/common must not depend on any other internal package").
		Dependencies().
		InPackage(module.Pkg("ui/common/...")).
		IsNotTest().
		DependOn(module.Pkgs("audio/...", "persistence/...", "render/...", "ui/synth/...", "ui/tracker/...")...).
		Test(t, ws)
}

// TestArch_PersistenceDoesNotDependOnUISynth enforces that the persistence
// layer stays decoupled from synth panel UI details.
func TestArch_PersistenceDoesNotDependOnUISynth(t *testing.T) {
	ws := loadWorkspace(t)

	archscout.Rule("persistence must not depend on ui/synth").
		Dependencies().
		InPackage(module.Pkg("persistence/...")).
		IsNotTest().
		DependOn(module.Pkg("ui/synth/...")).
		Test(t, ws)
}

// TestArch_NoPanicInProductionCode enforces that non-test production code
// never calls the built-in panic, keeping error handling explicit.
// The root main package is excluded — os.Exit in main() is legitimate.
func TestArch_NoPanicInProductionCode(t *testing.T) {
	ws := loadWorkspace(t)

	archscout.Rule("panic forbidden in production code").
		FunctionCalls().
		InPackage(module.Pkg("...")).
		NotInPackage(string(module)). // main() may call os.Exit
		IsNotTest().
		Match(func(fc archscout.FunctionCall) bool {
			return slices.Contains([]string{"panic", "os.Exit", "log.Fatal", "log.Fatalf"}, fc.Callee)
		}).
		Test(t, ws)
}

// TestArch_OnlyAudioOrRenderDependsOnOto enforces that the oto audio backend
// is only imported by the render package, which owns sinks and playback scheduling.
func TestArch_OnlyRenderDependsOnOto(t *testing.T) {
	ws := loadWorkspace(t)

	archscout.Rule("only render may depend on oto packages").
		Dependencies().
		InPackage(module.Pkg("...")).
		NotInPackage(module.Pkg("render/...")).
		IsNotTest().
		DependOn("github.com/ebitengine/oto/...").
		Test(t, ws)
}

// TestArch_AudioExportsPublicAPITypes verifies that the audio package exposes
// the key domain types the rest of the codebase depends on.
func TestArch_AudioExportsPublicAPITypes(t *testing.T) {
	ws := loadWorkspace(t)

	for _, typeName := range []string{"Synth", "Volume"} {
		archscout.Rule("audio must export type "+typeName).
			Types().
			InPackage(module.Pkg("audio")).
			Match(func(typ archscout.Type) bool { return typ.Name == typeName }).
			ShouldExist().
			Test(t, ws)
	}
}

// TestArch_AudioExportsStreamingTypes verifies that the audio package exposes
// the key streaming types the rest of the codebase depends on (now defined locally,
// not re-exported from beep).
func TestArch_AudioExportsStreamingTypes(t *testing.T) {
	ws := loadWorkspace(t)

	for _, typeName := range []string{"SampleRate", "Streamer"} {
		archscout.Rule("audio must export type "+typeName).
			Types().
			InPackage(module.Pkg("audio")).
			Match(func(typ archscout.Type) bool { return typ.Name == typeName }).
			ShouldExist().
			Test(t, ws)
	}
}

// TestArch_AudioVolumeConstructorExists verifies that the Volume constructor
// is present, enforcing the design where callers use NewVolume not a raw struct.
func TestArch_AudioVolumeConstructorExists(t *testing.T) {
	ws := loadWorkspace(t)

	archscout.Rule("audio must export NewVolume constructor").
		Functions().
		InPackage(module.Pkg("audio")).
		Match(func(fn archscout.Function) bool { return fn.Name == "NewVolume" }).
		ShouldExist().
		Test(t, ws)
}
