package inproc

// nfd returns the Unicode NFD (canonical decomposition + canonical ordering)
// of s. It uses the tables in nfd_tables.go so go-piper needs no external
// module (it replaces golang.org/x/text/unicode/norm). The behaviour matches
// x/text's norm.NFD for the whole code space (see the generator/tests).
func nfd(s string) string {
	// 1. Recursive canonical decomposition.
	b := make([]rune, 0, len(s))
	for _, r := range s {
		appendDecomposed(&b, r)
	}
	// 2. Canonical ordering: a stable sort of combining-mark runs by CCC.
	canonicalOrder(b)
	return string(b)
}

// appendDecomposed appends the full canonical decomposition of r to *b.
func appendDecomposed(b *[]rune, r rune) {
	if l, v, t, ok := hangulDecompose(r); ok {
		if l != 0 {
			*b = append(*b, l)
		}
		if v != 0 {
			*b = append(*b, v)
		}
		if t != 0 {
			*b = append(*b, t)
		}
		return
	}
	if d, ok := canonicalDecomp[r]; ok {
		*b = append(*b, d...)
		return
	}
	*b = append(*b, r)
}

// hangulDecompose splits a Hangul syllable into its L, V (and optional T) jamo,
// which are not part of the decomposition table (Unicode algorithm).
func hangulDecompose(r rune) (l, v, t rune, ok bool) {
	const (
		sBase  = 0xAC00
		lBase  = 0x1100
		vBase  = 0x1161
		tBase  = 0x11A7
		tCount = 28
		nCount = 588 // vCount * tCount
		sCount = 11172
	)
	if r < sBase || r >= sBase+sCount {
		return 0, 0, 0, false
	}
	si := int(r - sBase)
	l = rune(lBase + si/nCount)
	v = rune(vBase + (si%nCount)/tCount)
	if ti := si % tCount; ti != 0 {
		t = rune(tBase + ti)
	}
	return l, v, t, true
}

// canonicalOrder implements the UAX #15 canonical ordering algorithm: within
// every run of non-starters, characters are ordered by ascending combining
// class (stable).
func canonicalOrder(b []rune) {
	for i := 1; i < len(b); i++ {
		cc := combiningClass[b[i]]
		if cc == 0 {
			continue
		}
		for j := i; j > 0; j-- {
			pc := combiningClass[b[j-1]]
			if pc <= cc {
				break
			}
			b[j-1], b[j] = b[j], b[j-1]
		}
	}
}
