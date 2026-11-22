// Package notices is the messages panel: the notices it builds from the
// facts the root copies in, its cursor and body scroll, its keys and its
// frame. The root persists dismissals and opens links; it hands the panel
// copied facts and runs what its keys ask for.
package notices

import (
	"encoding/json"
	"strings"
	"time"
)

// IDs of the notices that ship in the binary.
const (
	ConfigNotImported = "config-not-imported"
)

const RepoURL = "https://github.com/Doremi203/aghelm"

// Kind groups notices in the list, in the order the groups are shown.
type Kind uint8

const (
	KindRelease Kind = iota
	KindNews
	KindHeadsUp
	KindTip
)

type Notice struct {
	ID       string
	Kind     Kind
	Subtitle string
	Title    string
	Accent   []string
	Body     []string
	URL      string
	Action   string
	Pinned   bool
	Expires  time.Time
	Unread   bool
}

func (n Notice) IsUpdate() bool {
	return strings.HasPrefix(n.ID, "update-")
}

const (
	DismissedSetting = "dismissed_notices"
	SeenSetting      = "seen_notices"
)

// SettingReader is the store as the panel's persisted sets read from it.
type SettingReader interface {
	Setting(key string) (string, error)
}

// Dismissed is the notice ids the user closed for good.
func Dismissed(st SettingReader) map[string]bool { return noticeIDs(st, DismissedSetting) }

// Seen is the notice ids already read.
func Seen(st SettingReader) map[string]bool { return noticeIDs(st, SeenSetting) }

func noticeIDs(st SettingReader, key string) map[string]bool {
	set := map[string]bool{}
	raw, err := st.Setting(key)
	if err != nil || raw == "" {
		return set
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return set
	}
	for _, id := range ids {
		set[id] = true
	}
	return set
}
