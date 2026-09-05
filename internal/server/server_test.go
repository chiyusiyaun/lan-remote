package server

import "testing"

func TestGeneratePIN(t *testing.T) {
	p := GeneratePIN()
	if len(p) != 6 {
		t.Fatalf("pin len=%d want 6", len(p))
	}
	for _, c := range p {
		if c < '0' || c > '9' {
			t.Fatalf("pin has non-digit %q", c)
		}
	}
	// uniqueness smoke
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		seen[GeneratePIN()] = true
	}
	if len(seen) < 2 {
		t.Fatal("pins look constant")
	}
}
