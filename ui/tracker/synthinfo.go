package tracker

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/common"
)

var modDestShortNames = []string{"Pitch", "Vol", "Cut", "PW", "Detune"}

var synthInfoLabelStyle = lipgloss.NewStyle().Foreground(common.ColorTextMuted)

func lbl(s string) string { return synthInfoLabelStyle.Render(s) }

func vlbl(s string, clr color.Color) string {
	return lipgloss.NewStyle().Foreground(clr).Render(s)
}

// isVoiceMuted reports whether a voice should be omitted from the synth info.
// A voice is considered muted when its oscillator type is Silent, its mixer
// channel is explicitly muted, or its mixer volume is zero.
func isVoiceMuted(osc audio.Oscillator, mixerVolume float64, mixerMute bool) bool {
	return osc.Type == audio.Silent || mixerMute || mixerVolume == 0
}

// renderSynthInfo returns a compact multi-line summary of the synth patch
// assigned to the current track. Voices whose oscillator is muted are omitted.
// Voice groups (Osc/Env/LFO) are color-coded to match the synth screen.
func renderSynthInfo(synth *audio.Synth) string {
	if synth == nil {
		return "(no synth)"
	}
	var sb strings.Builder
	if synth.Meta.Name != "" {
		fmt.Fprintf(&sb, "%s %s", lbl("Patch:"), synth.Meta.Name)
		sb.WriteString("\n")
		if synth.Meta.Bank != "" {
			fmt.Fprintf(&sb, "%s %s", lbl("Bank:"), synth.Meta.Bank)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	type voiceDef struct {
		osc  audio.Oscillator
		env  audio.Envelope
		lfo  audio.LFO
		vol  float64
		mute bool
		num  string
		clr  color.Color
	}
	voices := []voiceDef{
		{synth.Oscillator1, synth.Envelope1, synth.LFO1, synth.Mixer.Volume1, synth.Mixer.Mute1, "1", common.ColorAccentPrimary},
		{synth.Oscillator2, synth.Envelope2, synth.LFO2, synth.Mixer.Volume2, synth.Mixer.Mute2, "2", common.ColorAccentEnvelope},
		{synth.Oscillator3, synth.Envelope3, synth.LFO3, synth.Mixer.Volume3, synth.Mixer.Mute3, "3", common.ColorAccentPlay},
	}

	for _, v := range voices {
		if isVoiceMuted(v.osc, v.vol, v.mute) {
			continue
		}
		sb.WriteString(renderOscLine("Osc"+v.num, v.osc, v.clr))
		sb.WriteString("\n")
		sb.WriteString(renderEnvLine("Env"+v.num, v.env, v.clr))
		sb.WriteString("\n")
		sb.WriteString(renderLFOLine("LFO"+v.num, v.lfo, v.clr))
		sb.WriteString("\n\n")
	}
	sb.WriteString(renderMixLine(synth.Mixer))
	sb.WriteString("\n")
	sb.WriteString(renderFilterLine(synth.Filter))
	return sb.String()
}

func renderOscLine(label string, osc audio.Oscillator, clr color.Color) string {
	extra := ""
	if osc.Type == audio.Square {
		pw := osc.PulseWidth
		if pw == 0 {
			pw = 0.5
		}
		extra += fmt.Sprintf(" %s%d%%", lbl("PW:"), int(pw*100))
	}
	if osc.Detune != 0 {
		extra += fmt.Sprintf(" %s%+.0fc", lbl("Dtune:"), osc.Detune)
	}
	return fmt.Sprintf("%s %-16s%s", vlbl(label+":", clr), osc.Type, extra)
}

func renderEnvLine(label string, env audio.Envelope, clr color.Color) string {
	return fmt.Sprintf("%s %s%s %s%s %s%d%% %s%s",
		vlbl(label+":", clr),
		lbl("A:"), fmtDur(env.Attack),
		lbl("D:"), fmtDur(env.Decay),
		lbl("S:"), int(env.Sustain*100),
		lbl("R:"), fmtDur(env.Release),
	)
}

func renderLFOLine(label string, lfo audio.LFO, clr color.Color) string {
	if lfo.Depth == 0 {
		return fmt.Sprintf("%s (off)", vlbl(label+":", clr))
	}
	dest := modDestShortNames[int(lfo.Dest)%len(modDestShortNames)]
	return fmt.Sprintf("%s %-9s %.2fHz %d%% %s%s",
		vlbl(label+":", clr),
		lfo.Waveform,
		lfo.Rate,
		int(lfo.Depth*100),
		lbl("→"),
		dest,
	)
}

func renderMixLine(m audio.Mixer) string {
	return fmt.Sprintf("%s %s%d%% %s%d%% %s%d%%",
		lbl("Mix:"),
		lbl("Osc1:"), int(m.Volume1*100),
		lbl("Osc2:"), int(m.Volume2*100),
		lbl("Osc3:"), int(m.Volume3*100),
	)
}

func renderFilterLine(f audio.Filter) string {
	if f.Type == audio.FilterOff {
		return fmt.Sprintf("%s (off)", lbl("Filt:"))
	}
	return fmt.Sprintf("%s %-10s %s%d%% %s%d%%",
		lbl("Filt:"),
		f.Type,
		lbl("Cut:"), int(f.Cutoff*100),
		lbl("Res:"), int(f.Resonance*100),
	)
}

func fmtDur(d time.Duration) string {
	ms := d.Milliseconds()
	if ms == 0 {
		return "0  "
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
