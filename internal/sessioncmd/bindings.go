package sessioncmd

import (
	"github.com/Doremi203/aghelm/internal/store"
	"github.com/Doremi203/aghelm/internal/tmux"
)

// ApplyBindingSettings reads what the in-session bindings are built from.
// Every process that installs them reads it here, so none strips what
// another put on the server.
func ApplyBindingSettings(st *store.Store, driver *tmux.Driver) error {
	keys, err := st.SessionKeys()
	if err != nil {
		return err
	}
	driver.SetSessionKeys(keys)
	return nil
}
