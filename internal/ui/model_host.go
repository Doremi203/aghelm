package ui

// Host methods several feature types declare. Each feature's own host
// interface names the subset it uses.

func (m *Model) Size() (width, height int) { return m.layout.width, m.layout.height }
