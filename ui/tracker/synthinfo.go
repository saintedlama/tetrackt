package tracker

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/common"
)

var modDestShortNames = []string{"Pitch", "Vol", "Cut", "PW", "Detune"}

var synthInfoLabelStyle = lipgloss.NewStyle().Foreground(common.ColorTextMuted)

func lbl(s string) string { return synthInfoLabelStyle.Render(s) }

// renderSynthInfo returns a compact multi-line summary of the synth patch
// assigned to the current track.
func renderSynthInfo(synth *audio.Synth) string {
	if synth == nil {
		return "(no synth)"
	}
	var sb strings.Builder
	sb.WriteString(renderOscLine("Osc1", synth.Oscillator1))
	sb.WriteString("\n")
	sb.WriteString(renderOscLine("Osc2", synth.Oscillator2))
	sb.WriteString("\n")
	sb.WriteString(renderEnvLine("Env1", synth.Envelope1))
	sb.WriteString("\n")
	sb.WriteString(renderEnvLine("Env2", synth.Envelope2))
	sb.WriteString("\n")
	sb.WriteString(renderLFOLine("LFO1", synth.LFO1))
	sb.WriteString("\n")
	sb.WriteString(renderLFOLine("LFO2", synth.LFO2))
	sb.WriteString("\n")
	sb.WriteString(renderMixLine(synth.Mixer, synth.Portamento))
	sb.WriteString("\n")
	sb.WriteString(renderFilterLine(synth.Filter))
	return sb.String()
}

func renderOscLine(label string, osc audio.Oscillator) string {
	if osc.Type == audio.Silent {
		return fmt.Sprintf("%s (silent)", lbl(label+":"))
	}
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
	return fmt.Sprintf("%s %-16s%s", lbl(label+":"), osc.Type, extra)
}

func renderEnvLine(label string, env audio.Envelope) string {
	return fmt.Sprintf("%s %s%s %s%s %s%d%% %s%s",
		lbl(label+":"),
		lbl("A:"), fmtDur(env.Attack),
		lbl("D:"), fmtDur(env.Decay),
		lbl("S:"), int(env.Sustain*100),
		lbl("R:"), fmtDur(env.Release),
	)
}

func renderLFOLine(label string, lfo audio.LFO) string {
	if lfo.Depth == 0 {
		return fmt.Sprintf("%s (off)", lbl(label+":"))
	}
	dest := modDestShortNames[int(lfo.Dest)%len(modDestShortNames)]
	return fmt.Sprintf("%s %-9s %.2fHz %d%% %s%s",
		lbl(label+":"),
		lfo.Waveform,
		lfo.Rate,
		int(lfo.Depth*100),
		lbl("→"),
		dest,
	)
}

func renderMixLine(m audio.Mixer, portamento float64) string {
	s := fmt.Sprintf("%s %s%d%% %s%d%%",
		lbl("Mix:"),
		lbl("Osc1:"), int(m.Volume1*100),
		lbl("Osc2:"), int(m.Volume2*100),
	)
	if portamento > 0 {
		s += fmt.Sprintf(" %s%.2fs", lbl("Glide:"), portamento)
	}
	return s
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
