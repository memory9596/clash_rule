package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
)

type pdbIX struct {
	ID      uint32 `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

type pdbIXResponse struct {
	Data []pdbIX `json:"data"`
}

type pdbPfx struct {
	Prefix   string `json:"prefix"`
	IxlanID  uint32 `json:"ixlan_id"`
	Protocol string `json:"protocol"`
}

type pdbPfxResponse struct {
	Data []pdbPfx `json:"data"`
}

func fetchCNIXPrefixes(v4, v6 map[string]struct{}) error {
	fmt.Println("Fetching CN IXP prefixes from PeeringDB...")

	resp, err := http.Get("https://www.peeringdb.com/api/ix?country=CN")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var ixData pdbIXResponse
	if err := json.NewDecoder(resp.Body).Decode(&ixData); err != nil {
		return err
	}

	cnIxIDs := make(map[uint32]bool)
	for _, ix := range ixData.Data {
		cnIxIDs[ix.ID] = true
	}

	if len(cnIxIDs) == 0 {
		fmt.Println("No CN IXPs found in PeeringDB")
		return nil
	}

	pfxResp, err := http.Get("https://www.peeringdb.com/api/ixpfx")
	if err != nil {
		return err
	}
	defer pfxResp.Body.Close()

	var pfxData pdbPfxResponse
	if err := json.NewDecoder(pfxResp.Body).Decode(&pfxData); err != nil {
		return err
	}

	addedCount := 0
	for _, pfx := range pfxData.Data {
		if cnIxIDs[pfx.IxlanID] {
			p, err := netip.ParsePrefix(pfx.Prefix)

			if err != nil || p.Bits() == 0 {
				continue
			}

			// if isDoD(p) {
			// 	continue
			// }

			switch pfx.Protocol {
			case "IPv4":
				v4[pfx.Prefix] = struct{}{}
			case "IPv6":
				v6[pfx.Prefix] = struct{}{}
			}
			addedCount++
		}
	}

	fmt.Printf("Added %d IXP prefixes from PeeringDB\n", addedCount)
	return nil
}
