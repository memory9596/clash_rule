package main

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	start := time.Now()

	updateASNData()

	dataDir := "./data"
	os.MkdirAll(dataDir, 0755)

	// rvSources := map[string]string{
	// 	"005_route-views.amsix": "https://routeviews.org/amsix.ams/bgpdata/",
	// }
	// for name, base := range rvSources {
	// 	if url, err := getRouteViewsLatestURL(base); err == nil {
	// 		parts := strings.Split(url, "/")
	// 		fileName := parts[len(parts)-1]
	// 		downloadFile(url, filepath.Join(dataDir, name+"_"+fileName))
	// 	}
	// }

	rrcSources := map[string]string{
		"001_rrc14": "https://data.ris.ripe.net/rrc14/",
		"002_rrc21": "https://data.ris.ripe.net/rrc21/",
		"003_rrc12": "https://data.ris.ripe.net/rrc12/",
		"004_rrc23": "https://data.ris.ripe.net/rrc23/",
	}
	for name, base := range rrcSources {
		url := base + "latest-bview.gz"
		parts := strings.Split(url, "/")
		fileName := parts[len(parts)-1]
		downloadFile(url, filepath.Join(dataDir, name+"_"+fileName))
	}

	v4Routes, v6Routes := make(map[string]struct{}), make(map[string]struct{})
	files, _ := os.ReadDir(dataDir)

	for _, f := range files {
		if !f.IsDir() {
			processMRTFile(filepath.Join(dataDir, f.Name()), v4Routes, v6Routes)
		}
	}

	if err := fetchCNIXPrefixes(v4Routes, v6Routes); err != nil {
		fmt.Printf("Warning: failed to fetch IX prefixes: %v\n", err)
	}

	v4Final := aggregatePrefixes(v4Routes)
	v6Final := aggregatePrefixes(v6Routes)

	writePrefixesToFile("chnroutes.txt", v4Final)
	writePrefixesToFile("chnroutes6.txt", v6Final)

	var combined []netip.Prefix
	combined = append(combined, v4Final...)
	combined = append(combined, v6Final...)
	writePrefixesToFile("cn.list", combined)

	fmt.Printf("Task Complete. V4: %d, V6: %d\n", len(v4Final), len(v6Final))
	fmt.Printf("Total execution time: %s\n", time.Since(start))
}
