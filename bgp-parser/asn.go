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

// https://bgp.tools/kb/what-is-a-upstream
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

// https://bgp.tools/tags/cdn.csv
var cdn = map[uint32]bool{
	13335: false, // Cloudflare, Inc.
	395403:false, // IBM
	16276: false, // OVH SAS
	19551: false, // Incapsula Inc
	22822: false, // Edgio (Limelight)
	12989: false, // Black HOST Ltd
	33438: false, // Datum
	54113: false, // Fastly, Inc.
	36408: false, // CDNetworks Inc.
	30081: false, // CacheFly
	200325:false, // Bunny.net
	32934: false, // Facebook, Inc.
	60068: false, // Datacamp Limited (CDN77 / DataPacket)
	15169: false, // Google LLC
	46489: false, // Amazon IVS (Twitch)
	57976: false, // Blizzard Entertainment, Inc
	714:   false, // Apple Inc.
	40676: false, // Psychz Networks
	27647: false, // Weebly, Inc.
	31424: false, // Netrics AG
	49071: false, // Schweizerische Radio- und Fernsehgesellschaft
	16265: false, // LeaseWeb Network B.V.
	50069: false, // Misaka Network, Inc.
	22773: false, // Cox Communications Inc.
	29789: false, // Reflected Networks, Inc.
	16509: false, // Amazon.com, Inc.
	6185:  false, // Apple Inc.
	7195:  false, // EdgeUno
	47787: false, // EDGOO NETWORKS UNIPESSOAL LDA
	2906:  false, // Netflix Streaming Services Inc.
	328316:false, // Point d'Atterissement Virtuel - Burkina Faso
	199524:false, // Gcore S.A.
	34655: false, // JWE S.a r.l.
	396998:false, // Path Network, Inc.
	211757:false, // Panascais ehf.
	399531:false, // Hop, Inc
	210004:false, // Stichting Internet Domeinregistratie Nederland
	4134:  false, // China Telecom Backbone
	7671:  false, // NTT SmartConnect Corporation
	4766:  false, // Korea Telecom
	32590: false, // Valve Corporation
	58453: false, // China Mobile International
	32787: false, // Akamai (Prolexic)
	7922:  false, // Comcast Cable Communications, LLC
	59077: false, // Shanghai UCloud Information Technology Company Limited
	3462:  false, // HiNet (Chunghwa Telecom)
	59238: false, // Innovative Extremist Co., Ltd.
	14618: false, // Amazon.com, Inc.
	23693: false, // PT. Telekomunikasi Selular
	52580: false, // Azion Technologies Ltda.
	715:   false, // WoodyNet, Inc.
	42:    false, // WoodyNet, Inc.
	19281: false, // Quad9
	11019: false, // HAProxy Technologies, Inc.
	14840: false, // BR.DIGITAL 
	24429: false, // Alibaba Cloud
	208722:false, // Global DC Oy (Yandex Finland)
	37963: false, // Hangzhou Alibaba Advertising Co.,Ltd.
	57811: false, // Redge Technologies sp. z o.o.
	15133: false, // Edgecast Inc.
	20940: false, // Akamai International B.V.
	21859: false, // Zenlayer Inc
	45090: false, // Tencent Cloud
	20546: false, // SOPRADO GmbH
	63293: false, // Meta (CDN Network)
	270014:false, // GRUPO CG LIMITADA
	3214:  false, // xTom GmbH
	13414: false, // Twitter Inc.
	20253: false, // Qwilt Inc.
	197216:false, // Delta HighTech Ltd.
	51185: false, // MainStreaming SpA
	396986:false, // Bytedance, Inc. (TikTok)
	139341:false, // WeChat/QQ CDN
	2818:  false, // BBC
	54994: false, // Meteverse Limited
	39928: false, // Broadcasting TV Company Studio 1+1
	13238: false, // YANDEX LLC
	49477: false, // E-TF1 SAS
	138915:false, // Kaopu Cloud HK Limited
	38634: false, // DWANGO Co.,Ltd.
	25182: false, // Nederlandse Publieke Omroep
	49544: false, // i3D.net B.V
	1764:  false, // Next Layer Telecommunications
	139057:false, // Legend Dynasty Pte. Ltd.
	57277: false, // BAIKAL-IX (LLC Zero Kilometer)
	21395: false, // Orange Polska CDN
	55967: false, // Beijing Baidu Netcom Science and Technology Co., Ltd.
	6424:  false, // EDGOO NETWORKS
	33353: false, // Sony Interactive Entertainment LLC
	51324: false, // Exaring AG
	399647:false, // Rumble USA Inc.
	44907: false, // Telegram Messenger - CDN India
	62014: false, // Telegram Messenger - Singapore
	59930: false, // Telegram Messenger - Americas
	62041: false, // Telegram Messenger - Europe
	22697: false, // Roblox
	23344: false, // Disney Worldwide Services, Inc.
	208414:false, // WEDOS Internet, a.s.
	21245: false, // MEDIANOVA INTERNET HIZMETLERI VE TICARET ANONIM SIRKETI
	29264: false, // Canal+ Tech SAS
	10310: false, // Yahoo Holdings Inc.
	23576: false, // NAVER Cloud Corp.
	14907: false, // Wikimedia Foundation Inc.
	263807:false, // MEDIASTREAM SPA
	141013:false, // GetaFIX Content Network
	43500: false, // Warner Bros. Discovery
	23393: false, // NuCDN LLC
	136907:false, // Huawei Cloud
	36040: false, // Google (Global Cache)
	57363: false, // CDNvideo LLC
	21030: false, // Cluster LLC
}

var (
	cnEdge = make(map[uint32]bool)
	cnTag = make(map[uint32]bool)
)

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
				if _, exists := cdn[asn]; exists {
					cdn[asn] = true
				}
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

	if cdn[sec] && isOriginCnTag {
		return true
	}

	if cdn[origin] && isTier1(sec, isV6) {
		return true
	}

	return false
}
