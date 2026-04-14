package tracker

func (m *TrackerModel) cursorColumnLabel() string {
	switch m.CursorCol {
	case columnNote:
		return "NOTE"
	case columnVolume:
		return "VOL"
	case columnArpeggio:
		return "ARP"
	case columnEffect:
		return "FX"
	case columnParam:
		return "PARAM"
	default:
		return "?"
	}
}
