package unifi

import (
	"encoding/json"
	"fmt"
)

// metaResponse wraps (most) of the API response objects.
type metaResponse struct {
	Meta struct {
		RC      string `json:"rc"`  // "ok" on success, "error" on failure
		Message string `json:"msg"` // error message identificator

		// only on GET /status
		ServerVersion string `json:"server_version"`
		Up            bool   `json:"up"`
	} `json:"meta"`

	Data *json.RawMessage // usually an array of 1 object, sometimes empty
}

const loginPath = "/api/login"

// request for /api/login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
	Strict   bool   `json:"strict"`
}

const statusPath = "/status"

const sitesPath = "/api/self/sites"

type sitesResponse struct {
	ID   string `json:"_id"`  // BSON ObjectId
	Name string `json:"name"` // short letter code
	Desc string `json:"desc"` // human site name
}

const siteHealthPath = "/api/s/{siteName}/stat/widget/health"

type siteHealthResponse struct {
	AvgWifiUtilization struct {
		Band5  float64 `json:"na"`
		Band24 float64 `json:"ng"`
	} `json:"average_wifi_utilization"`

	WifiScore struct {
		ClientScoreAvg float64 `json:"client_score_avg"`
		PoorClients    int     `json:"clients_with_poor_score"`
		FairClients    int     `json:"clients_with_fair_score"`
		TotalClients   int     `json:"clients"`
	} `json:"wifi_score"`
}

const siteDevicesPath = "/api/s/{siteName}/stat/device"

type DeviceStatus int

var deviceStatus = []string{
	"disconnected",
	"connected",
	"pending",
	"firmware mismatch",
	"upgrading",
	"provisioning",
	"heartbeat missed",
	"adopting",
	"deleting",
	"inform error",
	"adoption failed",
	"isolated",
}

func (ds DeviceStatus) String() string {
	if int(ds) < len(deviceStatus) {
		return deviceStatus[ds]
	}

	return fmt.Sprintf("unknown (%d)", int(ds))
}

type siteDeviceResponse struct {
	MAC          string       `json:"mac"`
	Model        string       `json:"model"`   // short letter code
	Version      string       `json:"version"` // firmware version
	Adopted      bool         `json:"adopted"`
	LTS          bool         `json:"model_in_lts"`
	EOL          bool         `json:"model_in_eol"`
	State        DeviceStatus `json:"state"`
	LastSeenUnix int          `json:"last_seen"`
	Uptime       int          `json:"uptime"`

	Sys struct {
		Load1  string `json:"loadavg_1"`  // encoded as quoted float
		Load5  string `json:"loadavg_5"`  // dito
		Load15 string `json:"loadavg_15"` // dito
	} `json:"sys_stats"`

	Uplink struct {
		Type       string `json:"type"` // "wire", "wireless"
		FullDuplex bool   `json:"full_duplex"`
		Speed      int    `json:"speed"` // in MBit/s
	}

	// Virtual APs
	VAP []struct {
		Channel int    `json:"channel"`
		BSSID   string `json:"bssid"`
		ESSID   string `json:"essid"`
		Clients int    `json:"num_sta"`
		Radio   string `json:"radio"` // "na" (5GHz), "ng" (2.4GHz)
	} `json:"vap_table"`
}

func Band(radio string) string {
	switch radio {
	case "na":
		return "5"
	case "ng":
		return "2.4"
	default:
		return radio
	}
}

func (dev *siteDeviceResponse) ModelHuman() string {
	name, ok := modelLookup[dev.Model]
	if !ok {
		return "unknown"
	}
	return name
}

func (dev *siteDeviceResponse) UplinkDescription() string {
	switch u := dev.Uplink; u.Type {
	case "wire":
		dplx := "HD"
		if u.FullDuplex {
			dplx = "FD"
		}
		return fmt.Sprintf("%d%s", u.Speed, dplx)
	case "wireless":
		return "Mesh"
	default:
		return fmt.Sprintf("unknown (%s)", u.Type)
	}
}

func (dev *siteDeviceResponse) UplinkSpeed() int {
	if dev.Uplink.Type == "wire" {
		return dev.Uplink.Speed
	}
	return -1 // special case for wireless
}

var modelLookup = map[string]string{
	"BZ2":      "UniFi AP",
	"BZ2LR":    "UniFi AP-LR",
	"U2HSR":    "UniFi AP-Outdoor+",
	"U2IW":     "UniFi AP-In Wall",
	"U2L48":    "UniFi AP-LR",
	"U2Lv2":    "UniFi AP-LR v2",
	"U2M":      "UniFi AP-Mini",
	"U2O":      "UniFi AP-Outdoor",
	"U2S48":    "UniFi AP",
	"U2Sv2":    "UniFi AP v2",
	"U5O":      "UniFi AP-Outdoor 5G",
	"U7E":      "UniFi AP-AC",
	"U7EDU":    "UniFi AP-AC-EDU",
	"U7Ev2":    "UniFi AP-AC v2",
	"U7HD":     "UniFi AP-HD",
	"U7SHD":    "UniFi AP-SHD",
	"U7NHD":    "UniFi AP-nanoHD",
	"UFLHD":    "UniFi AP-Flex-HD",
	"UHDIW":    "UniFi AP-HD-In Wall",
	"UAIW6":    "U6-IW",
	"UAE6":     "U6-Extender",
	"UAL6":     "U6-Lite",
	"UAM6":     "U6-Mesh",
	"UALR6":    "U6-LR",
	"UAP6":     "U6-Pro",
	"UCXG":     "UniFi AP-XG",
	"UXSDM":    "UniFi AP-BaseStationXG",
	"UXBSDM":   "UniFi AP-BaseStationXG-Black",
	"UCMSH":    "UniFi AP-MeshXG",
	"U7IW":     "UniFi AP-AC-In Wall",
	"U7IWP":    "UniFi AP-AC-In Wall Pro",
	"U7MP":     "UniFi AP-AC-Mesh-Pro",
	"U7LR":     "UniFi AP-AC-LR",
	"U7LT":     "UniFi AP-AC-Lite",
	"U7O":      "UniFi AP-AC Outdoor",
	"U7P":      "UniFi AP-Pro",
	"U7MSH":    "UniFi AP-AC-Mesh",
	"U7PG2":    "UniFi AP-AC-Pro",
	"p2N":      "PicoStation M2",
	"UDMB":     "UniFi AP-BeaconHD",
	"USF5P":    "UniFi Switch Flex 5 POE",
	"US8":      "UniFi Switch 8",
	"US8P60":   "UniFi Switch 8 POE-60W",
	"US8P150":  "UniFi Switch 8 POE-150W",
	"S28150":   "UniFi Switch 8 AT-150W",
	"USC8":     "UniFi Switch 8",
	"USC8P60":  "UniFi Switch 8 POE-60W",
	"USC8P150": "UniFi Switch 8 POE-150W",
	"US16P150": "UniFi Switch 16 POE-150W",
	"S216150":  "UniFi Switch 16 AT-150W",
	"US24":     "UniFi Switch 24",
	"US24PRO":  "UniFi Switch PRO 24 POE",
	"US24PRO2": "UniFi Switch PRO 24",
	"US24P250": "UniFi Switch 24 POE-250W",
	"US24PL2":  "UniFi Switch 24 L2 POE",
	"US24P500": "UniFi Switch 24 POE-500W",
	"S224250":  "UniFi Switch 24 AT-250W",
	"S224500":  "UniFi Switch 24 AT-500W",
	"US48":     "UniFi Switch 48",
	"US48PRO":  "UniFi Switch PRO 48 POE",
	"US48PRO2": "UniFi Switch PRO 48",
	"US48P500": "UniFi Switch 48 POE-500W",
	"US48PL2":  "UniFi Switch 48 L2 POE",
	"US48P750": "UniFi Switch 48 POE-750W",
	"S248500":  "UniFi Switch 48 AT-500W",
	"S248750":  "UniFi Switch 48 AT-750W",
	"US6XG150": "UniFi Switch XG 6 POE",
	"USMINI":   "UniFi Switch Flex Mini",
	"USXG":     "UniFi Switch 16XG",
	"USC8P450": "UniFi Switch Industrial 8 POE-450W",
	"UDC48X6":  "UniFi Switch Leaf",
	"USL8A":    "UniFi Switch Aggregation",
	"USL8LP":   "UniFi Switch Lite 8 POE",
	"USL8MP":   "USW-Mission-Critical",
	"USL16P":   "UniFi Switch 16 POE",
	"USL16LP":  "UniFi Switch Lite 16 POE",
	"USL24":    "UniFi Switch 24",
	"USL48":    "UniFi Switch 48",
	"USL24P":   "UniFi Switch 24 POE",
	"USL48P":   "UniFi Switch 48 POE",
	"UGW3":     "UniFi Security Gateway 3P",
	"UGW4":     "UniFi Security Gateway 4P",
	"UGWHD4":   "UniFi Security Gateway HD",
	"UGWXG":    "UniFi Security Gateway XG-8",
	"UDM":      "UniFi Dream Machine",
	"UDMSE":    "UniFi Dream Machine SE",
	"UDMPRO":   "UniFi Dream Machine Pro",
	"UP4":      "UniFi Phone-X",
	"UP5":      "UniFi Phone",
	"UP5t":     "UniFi Phone-Pro",
	"UP7":      "UniFi Phone-Executive",
	"UP5c":     "UniFi Phone",
	"UP5tc":    "UniFi Phone-Pro",
	"UP7c":     "UniFi Phone-Executive",
	"UCK":      "UniFi Cloud Key",
	"UCK-v2":   "UniFi Cloud Key v2",
	"UCK-v3":   "UniFi Cloud Key v3",
	"UCKG2":    "UniFi Cloud Key Gen2",
	"UCKP":     "UniFi Cloud Key Gen2 Plus",
	"UASXG":    "UniFi Application Server XG",
	"ULTE":     "UniFi LTE",
	"ULTEPUS":  "UniFi LTE Pro",
	"ULTEPEU":  "UniFi LTE Pro",
	"UP1":      "UniFi Smart Power Plug",
	"UP6":      "UniFi Smart Power Strip",
	"USPRPS":   "UniFi Smart Power - Redundant Power System",
	"US624P":   "UniFi6 Switch 24",
	"UBB":      "UniFi Building Bridge",
	"UXGPRO":   "UniFi NeXt-Gen Gateway PRO",
}
