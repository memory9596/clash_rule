package main

import (
    "bufio"
    "compress/bzip2"
    "compress/gzip"
    "fmt"
    "io"
    "net/http"
    "net/netip"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
    "time"

    "github.com/osrg/gobgp/v3/pkg/packet/bgp"
    "github.com/osrg/gobgp/v3/pkg/packet/mrt"
)

const (
    // asnURL = "https://bgp.potaroo.net/cidr/autnums.html"
    asnURL = "https://ftp.ripe.net/ripe/asnames/asn.txt"
)

var cnBackbone = map[uint32]bool{
    4134: true,     // CHINANET-BACKBONE No.31,Jin-rong Street
    4837: true,     // CHINA169-BACKBONE CHINA UNICOM China169 Backbone
    9808: true,     // CHINAMOBILE-CN China Mobile Communications Group Co., Ltd.
    4538: true,     // ERX-CERNET-BKB China Education and Research Network Center
    4809: true,     // CHINATELECOM-CORE-WAN-CN2 China Telecom Next Generation Carrier Network
    23764: true,    // CTGNET CTGNet
    10099: true,    // UNICOM-GLOBAL China Unicom Global
    58453: true,    // CMI-INT-HK China Mobile International Limited
    58807: true,    // CMI-INT-AS China Mobile International Limited
    7497: true,     // CSTNET-AS-AP Computer Network Information Center of Chinese Academy of Sciences CNIC-CAS
    24151: true,    // CNNIC-CRITICAL-AP China Internet Network Infomation Center
    38345: true,    // ZDNS Internet Domain Name System Beijing Engineering Resrarch Center Ltd.
}

var cnIntl = map[uint32]bool{
    4809: true,     // CHINATELECOM-CORE-WAN-CN2 China Telecom Next Generation Carrier Network
    23764: true,    // CTGNET CTGNet
    10099: true,    // UNICOM-GLOBAL China Unicom Global
    58453: true,    // CMI-INT-HK China Mobile International Limited
    58807: true,    // CMI-INT-AS China Mobile International Limited
}

var cnProv = make(map[uint32]bool)
var cnTag = make(map[uint32]bool)

func downloadFile(url string, dest string) error {
    if _, err := os.Stat(dest); err == nil {
        fmt.Printf("File exists, skipping download: %s\n", dest)
        return nil
    }

    fmt.Printf("Downloading: %s\n", url)
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("http error: %s", resp.Status)
    }

    out, err := os.Create(dest)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    return err
}

func getRouteViewsLatestURL(baseURL string) (string, error) {
    now := time.Now().UTC()
    path := fmt.Sprintf("%04d.%02d/RIBS/", now.Year(), now.Month())
    fullURL := baseURL + path

    resp, err := http.Get(fullURL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    re := regexp.MustCompile(`rib\.\d{8}\.\d{4}\.bz2`)
    matches := re.FindAllString(string(body), -1)
    if len(matches) == 0 {
        return "", fmt.Errorf("no rib files found at %s", fullURL)
    }

    sort.Strings(matches)
    return fullURL + matches[len(matches)-1], nil
}

func updateASNData() {
    // dest := "AS Names.html"
    dest := "asn.txt"
    downloadFile(asnURL, dest)

    file, err := os.Open(dest)
    if err != nil {
        return
    }
    defer file.Close()

    // re := regexp.MustCompile(`AS(\d+)\s*</a>\s+(.*),\s+([A-Z]{2})$`)
    re := regexp.MustCompile(`^(\d+)\s+(.*),\s+([A-Z]{2})$`)
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }

        matches := re.FindStringSubmatch(line)
        if len(matches) == 4 {
            asnStr, country, desc := matches[1], matches[3], strings.ToLower(matches[2])
            if country == "CN" {
                var asn uint32
                fmt.Sscanf(asnStr, "%d", &asn)
                cnTag[asn] = true
                if !cnBackbone[asn] {
                    if strings.Contains(desc, "china telecom") ||
                        strings.Contains(desc, "chinatelecom") ||
                        strings.Contains(desc, "chinanet") ||
                        strings.Contains(desc, "china unicom") ||
                        strings.Contains(desc, "china mobile") ||
                        strings.Contains(desc, "cernet") ||
                        strings.Contains(desc, "cngi") ||
                        strings.Contains(desc, "china networks") ||
                        strings.Contains(desc, "cnnic") {
                        cnProv[asn] = true
                    }
                }
            }
        }
    }
    fmt.Printf("Updated ASN Database: Added %d CN tag and %d CN provincial backbone entries\n", len(cnTag), len(cnProv))
}

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

func main() {
    start := time.Now()

    updateASNData()

    dataDir := "./data"
    os.MkdirAll(dataDir, 0755)

    // rvSources := map[string]string{
    //     "005_route-views.amsix": "https://routeviews.org/amsix.ams/bgpdata/",
    // }

    // for name, base := range rvSources {
    //     url, err := getRouteViewsLatestURL(base)
    //     if err == nil {
    //         parts := strings.Split(url, "/")
    //         fileName := parts[len(parts)-1]
    //         dest := filepath.Join(dataDir, name+"_"+fileName)
    //         downloadFile(url, dest)
    //     }
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
        dest := filepath.Join(dataDir, name+"_"+fileName)
        downloadFile(url, dest)
    }

    v4Routes, v6Routes := make(map[string]struct{}), make(map[string]struct{})
    files, _ := os.ReadDir(dataDir)

    for _, f := range files {
        if !f.IsDir() {
            processMRTFile(filepath.Join(dataDir, f.Name()), v4Routes, v6Routes)
        }
    }

    v4Final := processAndMerge(v4Routes)
    v6Final := processAndMerge(v6Routes)

    writePrefixesToFile("chnroutes.txt", v4Final)
    writePrefixesToFile("chnroutes6.txt", v6Final)

    var combined []netip.Prefix
    combined = append(combined, v4Final...)
    combined = append(combined, v6Final...)
    writePrefixesToFile("cn.list", combined)

    elapsed := time.Since(start)

    fmt.Printf("Task Complete. V4: %d, V6: %d\n", len(v4Final), len(v6Final))

    fmt.Printf("Total execution time: %s\n", elapsed)
}

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

            if isDoD(p) {
                continue
            }

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
                    if matchRules(cleanPath(extractASList(asPath))) {
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

func matchRules(p []uint32) bool {
    n := len(p)
    if n == 0 {
        return false
    }

    origin := p[n-1]
    isOriginCnBackbone := cnBackbone[origin]
    isOriginCnIntl := cnIntl[origin]
    isOriginCnProv := cnProv[origin]
    isOriginCnTag := cnTag[origin]

    if isOriginCnIntl{
        return false
    }

    if isOriginCnBackbone || isOriginCnProv {
        return true
    }

    if n <= 2 {
        return false
    }

    sec := p[n-2]
    if cnProv[sec] {
        return true
    }
    if !cnIntl[sec] && cnBackbone[sec] && isOriginCnTag {
        return true
    }

    return false
}

func processAndMerge(routes map[string]struct{}) []netip.Prefix {
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
        w.WriteString(p.String() + "\n")
    }
    w.Flush()
}