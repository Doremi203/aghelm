package settings

import "testing"

func TestRestoreUsesTheSameDefaultsAsOpening(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"unset":    {DiffLayoutSetting: "", NotificationsSetting: ""},
		"enabled":  {DiffLayoutSetting: "split", NotificationsSetting: "on"},
		"disabled": {DiffLayoutSetting: "unified", NotificationsSetting: "off"},
	} {
		t.Run(name, func(t *testing.T) {
			want := FromSnapshot(Snapshot{Values: values})
			got := FromSnapshot(Snapshot{})
			var restored []Readback
			for key, raw := range values {
				restored = append(restored, Readback{Key: key, Value: raw})
			}
			got.Restore(restored, "", nil, nil)
			if got.LayoutSplit != want.LayoutSplit || got.Notifications != want.Notifications {
				t.Fatalf("restored layout/notifications = %v/%v, want %v/%v",
					got.LayoutSplit, got.Notifications, want.LayoutSplit, want.Notifications)
			}
		})
	}
}
