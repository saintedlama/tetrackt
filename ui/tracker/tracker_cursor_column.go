package tracker

func (m *TrackerModel) cursorColumnLabel() string {
	switch m.CursorCol {
	case columnVolume:
		return "VOL"
	case columnFX:
		return "FX"
	default:
		return "NOTE"
	}
}
