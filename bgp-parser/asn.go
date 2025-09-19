package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const asnURL = "https://ftp.ripe.net/ripe/asnames/asn.txt"

var cnBackbone = map[uint32]bool{
	4134:  true, // CHINANET-BACKBONE No.31,Jin-rong Street
	4837:  true, // CHINA169-BACKBONE CHINA UNICOM China169 Backbone
	9808:  true, // CHINAMOBILE-CN China Mobile Communications Group Co., Ltd.
	4538:  true, // ERX-CERNET-BKB China Education and Research Network Center
	7497:  true, // CSTNET-AS-AP Computer Network Information Center of Chinese Academy of Sciences CNIC-CAS
	24151: true, // CNNIC-CRITICAL-AP China Internet Network Infomation Center
	38345: true, // ZDNS Internet Domain Name System Beijing Engineering Resrarch Center Ltd.
}

var cnIntl = map[uint32]bool{
	4809:  true, // CHINATELECOM-CORE-WAN-CN2 China Telecom Next Generation Carrier Network
	23764: true, // CTGNET CTGNet
	10099: true, // UNICOM-GLOBAL China Unicom Global
	58453: true, // CMI-INT-HK China Mobile International Limited
	58807: true, // CMI-INT-AS China Mobile International Limited
}

var tier1 = map[uint32]bool{
	6762:  true, // Sparkle         // SEABONE-NET TELECOM ITALIA SPARKLE S.p.A., IT
	12956: true, // Telefonica      // TELXIUS TELEFONICA GLOBAL SOLUTIONS SL, ES
	2914:  true, // NTT             // NTT-DATA-2914 - NTT America, Inc., US
	3356:  true, // Lumen           // LEVEL3 - Level 3 Parent, LLC, US
	6453:  true, // TATA            // AS6453 - TATA COMMUNICATIONS (AMERICA) INC, US
	701:   true, // Verizon         // UUNET - Verizon Business, US
	6461:  true, // Zayo            // ZAYO-6461 - Zayo Bandwidth, US
	3257:  true, // GTT             // GTT-BACKBONE GTT Communications Inc., US
	1299:  true, // Telia           // TWELVE99 Arelion Sweden AB, SE
	3491:  true, // PCCW            // PCCWG-APAC-HK - PCCW Global (HK) Ltd., HK
	7018:  true, // AT&T            // ATT-INTERNET4 - AT&T Enterprises, LLC, US
	3320:  true, // DTAG            // DTAG Deutsche Telekom AG, DE
	5511:  true, // Orange          // Opentransit Orange S.A., FR
	6830:  true, // Liberty Global  // LibertyGlobal Liberty Global Europe Holding B.V., NL
	174:   true, // Cogent          // COGENT-174 - Cogent Communications, LLC, US
}

var cnEdge = make(map[uint32]bool)
var cnTag = make(map[uint32]bool)

func isTier1(asn uint32, isV6 bool) bool {
	if asn == 6939 { // HE (IPv6 Only)
		return isV6
	}
	return tier1[asn]
}

func updateASNData() {
	dest := "asn.txt"
	downloadFile(asnURL, dest)

	file, err := os.Open(dest)
	if err != nil {
		return
	}
	defer file.Close()

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
				if !cnBackbone[asn] && !cnIntl[asn] {
					if strings.Contains(desc, "china telecom") ||
						strings.Contains(desc, "chinatelecom") ||
						strings.Contains(desc, "chinanet") ||
						strings.Contains(desc, "china unicom") ||
						strings.Contains(desc, "china mobile") ||
						strings.Contains(desc, "mobile communication co") ||
						strings.Contains(desc, "mobile communications co") ||
						strings.Contains(desc, "china tietong") ||
						strings.Contains(desc, "cernet") ||
						strings.Contains(desc, "cngi") ||
						strings.Contains(desc, "china networks") ||
						strings.Contains(desc, "cnnic") ||
						strings.Contains(desc, "province network") ||
						strings.Contains(desc, "idc") ||
						strings.Contains(desc, "internet exchange point") ||
						strings.Contains(desc, "vpsor") ||
						strings.Contains(desc, "kaopu") ||
						strings.Contains(desc, "sakura") {
						cnEdge[asn] = true
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", dest, err)
	}
	fmt.Printf("Updated ASN Database: Added %d CN tag and %d CN edge network entries\n", len(cnTag), len(cnEdge))
}

func matchRules(p []uint32, isV6 bool) bool {
	n := len(p)
	if n == 0 {
		return false
	}

	origin := p[n-1]
	isOriginCnBackbone := cnBackbone[origin]
	isOriginCnIntl := cnIntl[origin]
	isOriginCnEdge := cnEdge[origin]
	isOriginCnTag := cnTag[origin]

	if isOriginCnIntl {
		return false
	}

	if isOriginCnBackbone || isOriginCnEdge {
		return true
	}

	if n < 2 {
		return false
	}

	sec := p[n-2]
	if cnEdge[sec] {
		return true
	}
	if !cnIntl[sec] && cnBackbone[sec] && isOriginCnTag {
		return true
	}
	if isTier1(sec, isV6) && isOriginCnTag {
		return true
	}

	return false
}
