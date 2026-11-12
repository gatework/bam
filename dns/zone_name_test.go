package dns

import (
	"strings"
	"testing"
)

func FuzzParseZoneName(f *testing.F) {
	for _, name := range []string{"ops.example.test", "OPS.Example.Test.", "example.test..", ".", "", ".test", "example..test", " test", "test\t", "münchen.example", "\xff.example"} {
		f.Add(name)
	}
	f.Fuzz(func(t *testing.T, input string) {
		name, err := parseZoneName(input)
		if err != nil {
			if name != (zoneName{}) {
				t.Fatal("invalid input retained a usable name")
			}
			return
		}
		canonical := name.value
		if canonical == "" || strings.HasPrefix(canonical, ".") || strings.HasSuffix(canonical, ".") || strings.Contains(canonical, "..") {
			t.Fatalf("normalization retained an empty label: %q", canonical)
		}
		if strings.ToLower(canonical) != canonical || strings.TrimSpace(canonical) != canonical {
			t.Fatalf("noncanonical lookup key: %q", canonical)
		}
		for _, equivalent := range []string{canonical, canonical + "."} {
			again, err := parseZoneName(equivalent)
			if err != nil || again != name {
				t.Fatalf("normalization is not idempotent: %q, %v", equivalent, err)
			}
		}
	})
}
