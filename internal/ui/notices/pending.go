package notices

// Log is what the root holds for the panel: the reason a config.toml of an
// earlier release was refused, shown as a notice until it is dismissed.
type Log struct {
	ImportError string
}
