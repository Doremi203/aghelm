package notices

import (
	"cmp"
	"slices"
)

// Sources are the root facts the notice list is built from, copied so the
// panel never reaches into the store or the key tables. ConfigImportError
// is why an old config.toml was refused.
type Sources struct {
	Store             bool
	ConfigImportError string
}

// Active is every notice the user has not dismissed, in display order.
func (p Panel) Active(src Sources) []Notice {
	if !src.Store {
		return nil
	}
	var notices []Notice
	if src.ConfigImportError != "" {
		notices = append(notices, Notice{
			ID:       ConfigNotImported,
			Kind:     KindHeadsUp,
			Subtitle: "Settings could not import it",
			Title:    "Your config.toml was not carried into Settings",
			Body: []string{
				"The keys and the editor live in Settings now. A config.toml that",
				"names them is read once to carry them over, and yours was refused:",
				"",
				src.ConfigImportError,
				"",
				"The keys are the defaults and the editor is picked for you until",
				"you set them in Settings. The file is not read again, and is yours",
				"to delete.",
			},
			URL:    RepoURL + "/blob/main/docs/configuration.md",
			Action: "Read what Settings holds",
		})
	}
	kept := notices[:0]
	for _, n := range notices {
		if !p.dismissed[n.ID] || n.Pinned {
			n.Unread = !p.seen[n.ID]
			kept = append(kept, n)
		}
	}
	slices.SortStableFunc(kept, func(a, b Notice) int {
		return cmp.Or(cmp.Compare(a.Kind, b.Kind), cmp.Compare(pinRank(a), pinRank(b)))
	})
	return kept
}

func pinRank(n Notice) int {
	if n.Pinned {
		return 0
	}
	return 1
}
