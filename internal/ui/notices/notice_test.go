package notices

import (
	"errors"
	"maps"
	"testing"
)

type settingsStub struct {
	values map[string]string
	err    error
}

func (s settingsStub) Setting(key string) (string, error) { return s.values[key], s.err }

func TestDismissedAndSeenReadTheirSettings(t *testing.T) {
	st := settingsStub{values: map[string]string{DismissedSetting: `["a","b"]`, SeenSetting: `["c"]`}}
	if got := Dismissed(st); !maps.Equal(got, map[string]bool{"a": true, "b": true}) {
		t.Fatalf("dismissed = %v", got)
	}
	if got := Seen(st); !maps.Equal(got, map[string]bool{"c": true}) {
		t.Fatalf("seen = %v", got)
	}
}

func TestDismissedIsEmptyOnAFailedOrBadRead(t *testing.T) {
	for name, st := range map[string]settingsStub{
		"error":    {values: map[string]string{DismissedSetting: `["a"]`}, err: errors.New("locked")},
		"bad json": {values: map[string]string{DismissedSetting: `[a`}},
		"unset":    {},
	} {
		if got := Dismissed(st); got == nil || len(got) != 0 {
			t.Fatalf("%s: dismissed = %v, want an empty set", name, got)
		}
	}
}
