package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/persistence"
	"github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/synth"
	"github.com/tetrackt/tetrackt/ui/tracker"
)

type mcpServer struct {
	bridge *mcpUIBridge
}

type setNotesArgs struct {
	Notes []noteUpdate `json:"notes"`
}

type noteUpdate struct {
	Track  int    `json:"track"`
	Row    int    `json:"row"`
	Note   string `json:"note"`
	Volume *int   `json:"volume,omitempty"`
}

type createPatchArgs struct {
	Name     string          `json:"name"`
	Category string          `json:"category,omitempty"`
	Tags     []string        `json:"tags,omitempty"`
	Synth    json.RawMessage `json:"synth"`
}

type assignPatchArgs struct {
	Track     int    `json:"track"`
	PatchName string `json:"patch_name"`
}

type applyCellEffectArgs struct {
	Track           int    `json:"track"`
	Row             int    `json:"row"`
	Effect          string `json:"effect"`
	Param           *int   `json:"param,omitempty"`
	VibratoSpeed    *int   `json:"vibrato_speed,omitempty"`
	VibratoDepth    *int   `json:"vibrato_depth,omitempty"`
	Ticks           *int   `json:"ticks,omitempty"`
	Continuous      *bool  `json:"continuous,omitempty"`
	ArpeggioOffsets []int  `json:"arpeggio_offsets,omitempty"`
}

type listPatchesArgs struct {
	BuiltinOnly bool `json:"builtin_only,omitempty"`
}

var trackerNotePattern = regexp.MustCompile(`^([A-G])(#?)-?([0-8])$`)

var noteBaseLookup = map[string]audio.Base{
	"C":  audio.BaseC,
	"C#": audio.BaseCs,
	"D":  audio.BaseD,
	"D#": audio.BaseDs,
	"E":  audio.BaseE,
	"F":  audio.BaseF,
	"F#": audio.BaseFs,
	"G":  audio.BaseG,
	"G#": audio.BaseGs,
	"A":  audio.BaseA,
	"A#": audio.BaseAs,
	"B":  audio.BaseB,
}

func runMCPServer(address string, bridge *mcpUIBridge) error {
	impl := &mcpServer{bridge: bridge}

	s := server.NewMCPServer(
		"tetrackt",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithInstructions("Controls the live TeTrackT UI session. MCP operations and user edits mutate the same in-memory tracker state. This server does not load or save song modules."),
	)

	s.AddTool(mcpToolTrackerInfo(), impl.handleTrackerInfo)
	s.AddTool(mcpToolSetNotes(), impl.handleSetNotes)
	s.AddTool(mcpToolCreatePatch(), impl.handleCreatePatch)
	s.AddTool(mcpToolListPatches(), impl.handleListPatches)
	s.AddTool(mcpToolAssignPatch(), impl.handleAssignPatch)
	s.AddTool(mcpToolSelectBuiltinPatch(), impl.handleSelectBuiltinPatch)
	s.AddTool(mcpToolApplyCellEffect(), impl.handleApplyCellEffect)

	httpServer := server.NewStreamableHTTPServer(s, server.WithEndpointPath("/mcp"))

	fmt.Printf("TeTrackT MCP server listening on http://%s/mcp\n", address)
	return httpServer.Start(address)
}

func (s *mcpServer) handleTrackerInfo(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := s.bridge.apply(ctx, func(m *model) (any, error) {
		tm := m.trackerModel()
		return map[string]any{
			"num_tracks": tm.NumTracks,
			"num_rows":   tm.NumRows,
			"bpm":        tm.BPM.Value(),
			"speed":      tm.Speed,
		}, nil
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("tracker_info failed", err), nil
	}
	return mcp.NewToolResultStructuredOnly(result), nil
}

func (s *mcpServer) handleSetNotes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args setNotesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultErrorFromErr("invalid arguments", err), nil
	}

	if len(args.Notes) == 0 {
		return mcp.NewToolResultError("notes cannot be empty"), nil
	}

	result, err := s.bridge.apply(ctx, func(m *model) (any, error) {
		tm := m.trackerModel()

		for _, change := range args.Notes {
			if err := validateTrackRow(tm, change.Track, change.Row); err != nil {
				return nil, err
			}

			note, err := parseTrackerNote(change.Note)
			if err != nil {
				return nil, err
			}

			cell := &tm.Tracks[change.Track].Rows[change.Row]
			cell.Note = note

			if change.Volume != nil {
				if *change.Volume < 0 || *change.Volume > 64 {
					return nil, fmt.Errorf("volume must be in range 0..64")
				}
				cell.Volume = *change.Volume
			}
		}

		m.dirty = true
		return map[string]any{"updated": len(args.Notes)}, nil
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("tracker_set_notes failed", err), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func (s *mcpServer) handleCreatePatch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args createPatchArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultErrorFromErr("invalid arguments", err), nil
	}

	if strings.TrimSpace(args.Name) == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	if len(args.Synth) == 0 {
		return mcp.NewToolResultError("synth payload is required"), nil
	}

	var synthPayload persistence.SavedSynth
	if err := json.Unmarshal(args.Synth, &synthPayload); err != nil {
		return mcp.NewToolResultErrorFromErr("invalid synth payload", err), nil
	}

	result, err := s.bridge.apply(ctx, func(m *model) (any, error) {
		patch := persistence.SavedPatch{
			Name:     strings.TrimSpace(args.Name),
			Category: strings.TrimSpace(args.Category),
			Tags:     ensureCustomTag(args.Tags),
			Synth:    synthPayload,
		}

		replaced := false
		for i := range m.bank.SynthPatches {
			if strings.EqualFold(m.bank.SynthPatches[i].Name, patch.Name) {
				m.bank.SynthPatches[i] = patch
				replaced = true
				break
			}
		}
		if !replaced {
			m.bank.SynthPatches = append(m.bank.SynthPatches, patch)
		}

		if err := m.bank.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "patch bank save failed: %v\n", err)
		}

		m.synth().SetUserPatches(bankToPatches(m.bank))
		m.dirty = true

		return map[string]any{
			"name":      patch.Name,
			"category":  patch.Category,
			"tags":      patch.Tags,
			"is_custom": true,
		}, nil
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("synth_create_patch failed", err), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func (s *mcpServer) handleListPatches(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listPatchesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultErrorFromErr("invalid arguments", err), nil
	}

	result, err := s.bridge.apply(ctx, func(m *model) (any, error) {
		all := m.synth().PatchBankView().Patches
		patches := make([]map[string]any, 0, len(all))
		for _, p := range all {
			if args.BuiltinOnly && p.IsCustom() {
				continue
			}

			patches = append(patches, map[string]any{
				"name":      p.Name,
				"category":  p.Category,
				"tags":      p.Tags,
				"is_custom": p.IsCustom(),
			})
		}

		return map[string]any{"patches": patches, "count": len(patches)}, nil
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("patchbank_list_patches failed", err), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func (s *mcpServer) handleAssignPatch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args assignPatchArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultErrorFromErr("invalid arguments", err), nil
	}

	result, err := s.bridge.apply(ctx, func(m *model) (any, error) {
		patch, err := findPatchByName(m.synth().PatchBankView().Patches, args.PatchName, false)
		if err != nil {
			return nil, err
		}

		if err := assignPatchToTrack(m, args.Track, patch); err != nil {
			return nil, err
		}

		m.dirty = true
		return map[string]any{
			"track":      args.Track,
			"patch_name": patch.Name,
			"category":   patch.Category,
			"tags":       patch.Tags,
		}, nil
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("track_assign_patch failed", err), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func (s *mcpServer) handleSelectBuiltinPatch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args assignPatchArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultErrorFromErr("invalid arguments", err), nil
	}

	result, err := s.bridge.apply(ctx, func(m *model) (any, error) {
		patch, err := findPatchByName(m.synth().PatchBankView().Patches, args.PatchName, true)
		if err != nil {
			return nil, err
		}

		if err := assignPatchToTrack(m, args.Track, patch); err != nil {
			return nil, err
		}

		m.dirty = true
		return map[string]any{
			"track":      args.Track,
			"patch_name": patch.Name,
			"category":   patch.Category,
			"tags":       patch.Tags,
		}, nil
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("track_select_builtin_patch failed", err), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func (s *mcpServer) handleApplyCellEffect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args applyCellEffectArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultErrorFromErr("invalid arguments", err), nil
	}

	effectType, err := parseEffectType(args.Effect)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := s.bridge.apply(ctx, func(m *model) (any, error) {
		tm := m.trackerModel()
		if err := validateTrackRow(tm, args.Track, args.Row); err != nil {
			return nil, err
		}

		row := &tm.Tracks[args.Track].Rows[args.Row]

		if args.Ticks != nil {
			if *args.Ticks < 1 || *args.Ticks > 32 {
				return nil, fmt.Errorf("ticks must be in range 1..32")
			}
			row.Ticks = *args.Ticks
		}

		if args.Continuous != nil {
			row.Continuous = *args.Continuous
		}

		if args.ArpeggioOffsets != nil {
			row.Arpeggio = audio.ArpeggioEffect{Offsets: append([]int(nil), args.ArpeggioOffsets...)}
		}

		effectParam, err := buildEffectParam(effectType, args)
		if err != nil {
			return nil, err
		}

		row.Effect = tracker.TrackerEffect{Type: effectType, Param: effectParam}
		m.dirty = true

		return map[string]any{
			"track":        args.Track,
			"row":          args.Row,
			"effect":       args.Effect,
			"effect_param": effectParam,
		}, nil
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("tracker_apply_cell_effect failed", err), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func validateTrackRow(tm *tracker.TrackerModel, trackIdx, rowIdx int) error {
	if trackIdx < 0 || trackIdx >= tm.NumTracks {
		return fmt.Errorf("track must be in range 0..%d", tm.NumTracks-1)
	}
	if rowIdx < 0 || rowIdx >= tm.NumRows {
		return fmt.Errorf("row must be in range 0..%d", tm.NumRows-1)
	}
	return nil
}

func assignPatchToTrack(m *model, trackIdx int, patch synth.SynthPatch) error {
	tm := m.trackerModel()
	if trackIdx < 0 || trackIdx >= tm.NumTracks {
		return fmt.Errorf("track must be in range 0..%d", tm.NumTracks-1)
	}

	tm.Tracks[trackIdx].Synth = cloneSynth(patch.Synth)
	if trackIdx == tm.CursorTrack() {
		m.synth().ApplyTrackChange(ui.TrackChanged{Synth: tm.Tracks[trackIdx].Synth})
	}
	return nil
}

func findPatchByName(patches []synth.SynthPatch, name string, builtinOnly bool) (synth.SynthPatch, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return synth.SynthPatch{}, fmt.Errorf("patch_name is required")
	}

	for _, p := range patches {
		if !strings.EqualFold(p.Name, name) {
			continue
		}
		if builtinOnly && p.IsCustom() {
			continue
		}
		return p, nil
	}

	if builtinOnly {
		return synth.SynthPatch{}, fmt.Errorf("builtin patch %q not found", name)
	}
	return synth.SynthPatch{}, fmt.Errorf("patch %q not found", name)
}

func ensureCustomTag(tags []string) []string {
	out := make([]string, 0, len(tags)+1)
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}

	if !slices.Contains(out, "Custom") {
		out = append([]string{"Custom"}, out...)
	}

	return out
}

func parseTrackerNote(value string) (audio.Note, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return audio.Note{}, fmt.Errorf("note is required")
	}

	if normalized == "---" || normalized == "OFF" {
		return audio.Off(), nil
	}

	match := trackerNotePattern.FindStringSubmatch(normalized)
	if len(match) != 4 {
		return audio.Note{}, fmt.Errorf("invalid note %q (expected C-4, C#4, or ---)", value)
	}

	baseName := match[1] + match[2]
	base, ok := noteBaseLookup[baseName]
	if !ok {
		return audio.Note{}, fmt.Errorf("invalid note base %q", baseName)
	}

	octave, err := strconv.Atoi(match[3])
	if err != nil {
		return audio.Note{}, fmt.Errorf("invalid octave in %q", value)
	}

	return audio.NewNote(base, audio.Octave(octave)), nil
}

func parseEffectType(value string) (tracker.EffectType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none":
		return tracker.EffectNone, nil
	case "vibrato":
		return tracker.EffectVibrato, nil
	case "volume_slide":
		return tracker.EffectVolumeSlide, nil
	case "note_cut":
		return tracker.EffectNoteCut, nil
	case "note_delay":
		return tracker.EffectNoteDelay, nil
	default:
		return tracker.EffectNone, fmt.Errorf("unsupported effect %q", value)
	}
}

func buildEffectParam(effectType tracker.EffectType, args applyCellEffectArgs) (int, error) {
	switch effectType {
	case tracker.EffectNone:
		return 0, nil
	case tracker.EffectVibrato:
		if args.Param != nil {
			if *args.Param < 0 || *args.Param > 255 {
				return 0, fmt.Errorf("param for vibrato must be in range 0..255")
			}
			return *args.Param, nil
		}

		speed := 4
		depth := 0
		if args.VibratoSpeed != nil {
			speed = *args.VibratoSpeed
		}
		if args.VibratoDepth != nil {
			depth = *args.VibratoDepth
		}

		if speed < 1 || speed > 15 {
			return 0, fmt.Errorf("vibrato_speed must be in range 1..15")
		}
		if depth < 0 || depth > 15 {
			return 0, fmt.Errorf("vibrato_depth must be in range 0..15")
		}

		return (speed << 4) | (depth & 0xF), nil

	case tracker.EffectVolumeSlide:
		param := 0
		if args.Param != nil {
			param = *args.Param
		}
		if param < -16 || param > 16 {
			return 0, fmt.Errorf("param for volume_slide must be in range -16..16")
		}
		return param, nil

	case tracker.EffectNoteCut, tracker.EffectNoteDelay:
		param := 0
		if args.Param != nil {
			param = *args.Param
		}
		if param < 0 || param > 31 {
			return 0, fmt.Errorf("param for %s must be in range 0..31", strings.ToLower(strings.TrimSpace(args.Effect)))
		}
		return param, nil

	default:
		return 0, fmt.Errorf("unsupported effect type")
	}
}

func cloneSynth(src *audio.Synth) *audio.Synth {
	if src == nil {
		return nil
	}

	saved := persistence.ToSavedSynth(src)
	return persistence.SynthFromSavedPatch(persistence.SavedPatch{Synth: saved})
}

func mcpToolTrackerInfo() mcp.Tool {
	return mcp.NewTool(
		"tracker_info",
		mcp.WithDescription("Get tracker dimensions and playback settings from the live UI state"),
	)
}

func mcpToolSetNotes() mcp.Tool {
	return mcp.NewTool(
		"tracker_set_notes",
		mcp.WithDescription("Set notes (and optional volume) for one or more tracker cells in the live session"),
		mcp.WithArray(
			"notes",
			mcp.Required(),
			mcp.Description("List of note updates"),
			mcp.MinItems(1),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"track": map[string]any{"type": "integer", "minimum": 0},
					"row":   map[string]any{"type": "integer", "minimum": 0},
					"note": map[string]any{
						"type":        "string",
						"description": "Tracker note token. Examples: C-4, C#4, A3, OFF, ---. Octave range: 0..8.",
						"pattern":     `^(---|OFF|[A-G]#?-?[0-8])$`,
					},
					"volume": map[string]any{"type": "integer", "minimum": 0, "maximum": 64},
				},
				"required": []string{"track", "row", "note"},
			}),
		),
	)
}

func mcpToolCreatePatch() mcp.Tool {
	return mcp.NewTool(
		"synth_create_patch",
		mcp.WithDescription("Create or replace a custom synth patch in the current UI session patch bank"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Patch name")),
		mcp.WithString("category", mcp.Description("Optional patch category")),
		mcp.WithArray("tags", mcp.WithStringItems(), mcp.Description("Optional tags")),
		mcp.WithAny("synth", mcp.Required(), mcp.Description("Synth payload matching persistence.SavedSynth fields")),
	)
}

func mcpToolListPatches() mcp.Tool {
	return mcp.NewTool(
		"patchbank_list_patches",
		mcp.WithDescription("List available patches from the current patch bank view"),
		mcp.WithBoolean("builtin_only", mcp.Description("Only include built-in patches")),
	)
}

func mcpToolAssignPatch() mcp.Tool {
	return mcp.NewTool(
		"track_assign_patch",
		mcp.WithDescription("Assign a built-in or custom patch to a track in the live tracker"),
		mcp.WithNumber("track", mcp.Required(), mcp.Description("0-based track index"), mcp.Min(0)),
		mcp.WithString("patch_name", mcp.Required(), mcp.Description("Patch name")),
	)
}

func mcpToolSelectBuiltinPatch() mcp.Tool {
	return mcp.NewTool(
		"track_select_builtin_patch",
		mcp.WithDescription("Assign a built-in patch from the patch bank to a track"),
		mcp.WithNumber("track", mcp.Required(), mcp.Description("0-based track index"), mcp.Min(0)),
		mcp.WithString("patch_name", mcp.Required(), mcp.Description("Built-in patch name")),
	)
}

func mcpToolApplyCellEffect() mcp.Tool {
	return mcp.NewTool(
		"tracker_apply_cell_effect",
		mcp.WithDescription("Apply tracker effects to a single cell in the live session"),
		mcp.WithNumber("track", mcp.Required(), mcp.Description("0-based track index"), mcp.Min(0)),
		mcp.WithNumber("row", mcp.Required(), mcp.Description("0-based row index"), mcp.Min(0)),
		mcp.WithString("effect", mcp.Required(), mcp.Enum("none", "vibrato", "volume_slide", "note_cut", "note_delay")),
		mcp.WithNumber("param", mcp.Description("Generic effect parameter")),
		mcp.WithNumber("vibrato_speed", mcp.Description("Vibrato speed 1..15"), mcp.Min(1), mcp.Max(15)),
		mcp.WithNumber("vibrato_depth", mcp.Description("Vibrato depth 0..15"), mcp.Min(0), mcp.Max(15)),
		mcp.WithNumber("ticks", mcp.Description("Row ticks 1..32"), mcp.Min(1), mcp.Max(32)),
		mcp.WithBoolean("continuous", mcp.Description("Continuous row synthesis")),
		mcp.WithArray("arpeggio_offsets", mcp.Description("Optional semitone offsets"), mcp.WithNumberItems()),
	)
}
