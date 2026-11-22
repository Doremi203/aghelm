package version

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in    string
		want  [3]int
		valid bool
	}{
		{"v0.8.2", [3]int{0, 8, 2}, true},
		{"0.8.2", [3]int{0, 8, 2}, true},
		{"v1.20.30", [3]int{1, 20, 30}, true},
		{"v0.9.0-12-gabc", [3]int{0, 9, 0}, true},
		{"dev", [3]int{}, false},
		{"v0.8", [3]int{}, false},
		{"v0.8.x", [3]int{}, false},
		{"", [3]int{}, false},
	}
	for _, test := range cases {
		got, ok := parseVersion(test.in)
		if ok != test.valid || got != test.want {
			t.Errorf("parseVersion(%q) = %v,%v want %v,%v", test.in, got, ok, test.want, test.valid)
		}
	}
}

func TestIsRelease(t *testing.T) {
	cases := map[string]bool{
		"0.47.1":                     true,
		"v0.47.1":                    true,
		"dev":                        false,
		"v0.47.1-10-g1656b508":       false,
		"v0.47.1-10-g1656b508-dirty": false,
		"v0.47.1-0-g1656b508":        false,
		"1656b508":                   false,
		"v0.47.2-0.20261106234132-2bf7711b3b5d+dirty": false,
		"": false,
	}
	for in, want := range cases {
		if got := IsRelease(in); got != want {
			t.Errorf("IsRelease(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestVersionWithin(t *testing.T) {
	if !VersionWithin("v0.8.2", "v0.8.0", "v0.9.0") {
		t.Fatal("version should fall inside inclusive bounds")
	}
	if VersionWithin("v0.8.2", "v0.8.3", "") {
		t.Fatal("version below minimum should not match")
	}
	if VersionWithin("dev", "v0.8.0", "") {
		t.Fatal("dev builds should not match targeted ranges")
	}
}
