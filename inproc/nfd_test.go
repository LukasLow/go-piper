package inproc

import "testing"

func TestNFDKnown(t *testing.T) {
	cases := []struct{ in, want string }{
		{"ç", "c\u0327"},                  // c-cedilla -> c + combining cedilla
		{"Å", "A\u030A"},                  // A-ring -> A + combining ring above
		{"héllo", "he\u0301llo"},          // é -> e + combining acute
		{"\u1E0B", "d\u0307"},             // ḋ -> d + combining dot above
		{"\uAC00", "\u1100\u1161"},        // Hangul LV (가)
		{"\uAC01", "\u1100\u1161\u11A8"},  // Hangul LVT (각)
		{"abc", "abc"},
	}
	for _, c := range cases {
		if got := nfd(c.in); got != c.want {
			t.Errorf("nfd(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNFDIdempotent(t *testing.T) {
	for _, s := range []string{"ç", "ǻ", "Hallo ç Å", "a\u0301\u0327"} {
		if a, b := nfd(s), nfd(nfd(s)); a != b {
			t.Errorf("nfd not idempotent: %q -> %q -> %q", s, a, b)
		}
	}
}
