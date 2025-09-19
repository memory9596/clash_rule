package main

import (
	"net/netip"
)

func isDoD(p netip.Prefix) bool {
	if !p.Addr().Is4() {
		return false
	}
	b := p.Addr().As4()
	switch b[0] {
	case 6, 7, 11, 21, 22, 26, 28, 29, 30, 33, 55, 214, 215:
		return true
	}
	return false
}
