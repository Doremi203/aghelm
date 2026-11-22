// Package version compares the semantic versions of aghelm builds and of the
// CLIs it launches.
package version

import (
	"strconv"
	"strings"
)

// parseVersion turns "v0.8.2" or "0.8.2" into its three numeric parts.
// A dev build or any tag without three numeric components returns ok=false.
func parseVersion(v string) ([3]int, bool) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	fields := strings.Split(v, ".")
	if len(fields) != 3 {
		return [3]int{}, false
	}
	var parts [3]int
	for i, field := range fields {
		n, err := strconv.Atoi(field)
		if err != nil {
			return [3]int{}, false
		}
		parts[i] = n
	}
	return parts, true
}

// IsRelease reports whether v names a published release, such as "0.47.1".
// A dev build and a git describe stamp such as "v0.47.1-10-g1656b508-dirty"
// do not.
func IsRelease(v string) bool {
	_, ok := parseVersion(v)
	return ok && !strings.ContainsAny(v, "-+")
}

// VersionWithin reports whether version falls inside the inclusive
// [minimum, maximum] range. An empty bound is open.
func VersionWithin(version, minimum, maximum string) bool {
	if minimum == "" && maximum == "" {
		return true
	}
	parts, ok := parseVersion(version)
	if !ok {
		return false
	}
	if minimum != "" {
		if low, ok := parseVersion(minimum); !ok || greater(low, parts) {
			return false
		}
	}
	if maximum != "" {
		if high, ok := parseVersion(maximum); !ok || greater(parts, high) {
			return false
		}
	}
	return true
}

func greater(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}
