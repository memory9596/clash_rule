package main

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strings"

	"github.com/osrg/gobgp/v3/pkg/packet/bgp"
	"github.com/osrg/gobgp/v3/pkg/packet/mrt"
)

func processMRTFile(path string, v4, v6 map[string]struct{}) {
	fmt.Printf("Processing: %s\n", path)
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	var r io.Reader = file
	if strings.HasSuffix(path, ".gz") {
		zr, _ := gzip.NewReader(file)
		defer zr.Close()
		r = zr
	} else if strings.HasSuffix(path, ".bz2") {
		r = bzip2.NewReader(file)
	}

	br := bufio.NewReader(r)
	hBuf := make([]byte, 12)

	for {
		if _, err := io.ReadFull(br, hBuf); err != nil {
			break
		}
		h := &mrt.MRTHeader{}
		if err := h.DecodeFromBytes(hBuf); err != nil {
			continue
		}
		bBuf := make([]byte, h.Len)
		if _, err := io.ReadFull(br, bBuf); err != nil {
			break
		}
		msg, err := mrt.ParseMRTBody(h, bBuf)
		if err != nil {
			continue
		}

		if rib, ok := msg.Body.(*mrt.Rib); ok {
			prefixStr := rib.Prefix.String()
			p, err := netip.ParsePrefix(prefixStr)

			if err != nil || p.Bits() == 0 {
				continue
			}

			// if isDoD(p) {
			// 	continue
			// }

			matched := false
			for _, entry := range rib.Entries {
				isIGP := false
				var asPath *bgp.PathAttributeAsPath
				for _, attr := range entry.PathAttributes {
					switch a := attr.(type) {
					case *bgp.PathAttributeOrigin:
						if a.Value == 0 {
							isIGP = true
						}
					case *bgp.PathAttributeAsPath:
						asPath = a
					}
				}
				if isIGP && asPath != nil {
					if matchRules(cleanPath(extractASList(asPath)), p.Addr().Is6()) {
						matched = true
						break
					}
				}
			}
			if matched {
				p := rib.Prefix.String()
				if strings.Contains(p, ":") {
					v6[p] = struct{}{}
				} else {
					v4[p] = struct{}{}
				}
			}
		}
	}
}

func extractASList(asPath *bgp.PathAttributeAsPath) []uint32 {
	var asList []uint32
	for _, param := range asPath.Value {
		if asParam, ok := param.(*bgp.As4PathParam); ok {
			if asParam.Type == bgp.BGP_ASPATH_ATTR_TYPE_SEQ {
				asList = append(asList, asParam.AS...)
			}
		}
	}
	return asList
}

func cleanPath(p []uint32) []uint32 {
	if len(p) == 0 {
		return p
	}
	res := []uint32{p[0]}
	for i := 1; i < len(p); i++ {
		if p[i] != res[len(res)-1] {
			res = append(res, p[i])
		}
	}
	return res
}
