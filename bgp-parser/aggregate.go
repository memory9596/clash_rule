package main

import (
	"bufio"
	"net/netip"
	"os"
	"sort"
)

func aggregatePrefixes(routes map[string]struct{}) []netip.Prefix {
	var prefixes []netip.Prefix
	for k := range routes {
		if p, err := netip.ParsePrefix(k); err == nil {
			prefixes = append(prefixes, p.Masked())
		}
	}
	sort.Slice(prefixes, func(i, j int) bool {
		a, b := prefixes[i], prefixes[j]
		if cmp := a.Addr().Compare(b.Addr()); cmp != 0 {
			return cmp < 0
		}
		return a.Bits() < b.Bits()
	})
	var merged []netip.Prefix
	for _, p := range prefixes {
		if len(merged) == 0 {
			merged = append(merged, p)
			continue
		}
		last := merged[len(merged)-1]
		if last.Overlaps(p) {
			continue
		}
		merged = append(merged, p)
		for len(merged) >= 2 {
			a, b := merged[len(merged)-2], merged[len(merged)-1]
			if a.Bits() == b.Bits() {
				sa, sb := supernet(a), supernet(b)
				if sa == sb && sa.IsValid() {
					merged = merged[:len(merged)-2]
					merged = append(merged, sa)
					continue
				}
			}
			break
		}
	}
	return merged
}

func supernet(p netip.Prefix) netip.Prefix {
	if p.Bits() == 0 {
		return netip.Prefix{}
	}
	return netip.PrefixFrom(p.Addr(), p.Bits()-1).Masked()
}

func writePrefixesToFile(filename string, prefixes []netip.Prefix) {
	f, _ := os.Create(filename)
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, p := range prefixes {
		_, _ = w.WriteString(p.String() + "\n")
	}
	w.Flush()
}
