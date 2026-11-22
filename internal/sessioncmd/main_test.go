package sessioncmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/Doremi203/aghelm/internal/testenv"
	"github.com/Doremi203/aghelm/internal/tmux"
)

// TestMain drops an inherited GIT_DIR and its siblings, which would point
// the temp repos these tests build at the repo running them.
func TestMain(m *testing.M) {
	tmux.ConfigFile = os.DevNull
	if err := testenv.IsolateGit(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
