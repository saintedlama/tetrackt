package tracker

func (m *TrackerModel) cursorColumnLabel() string {
	switch m.CursorCol {
	case ColumnNote:
		return "Note"
	case ColumnVolume:
		return "Volume"
	case ColumnArpeggio:
		return "Arp"
	case ColumnEffect:
		return "Fx"
	case ColumnParam:
		return "Param"
	default:
		return "?"
	}
}
