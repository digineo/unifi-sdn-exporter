package unifi

import "time"

type Metrics struct {
	ControllerVersion string

	AvgWifiUtilization24 float64
	AvgWifiUtilization50 float64
	AvgWifiScore         float64

	ClientsPoorScore int
	ClientsFairScore int
	ClientsGoodScore int

	Devices []DeviceMetrics
}

type DeviceMetrics struct {
	MAC        string
	Firmware   string
	Model      string
	ModelHuman string
	Adopted    bool
	LTS, EOL   bool

	Status      int
	StatusHuman string

	Uptime      time.Duration
	LastSeen    *time.Time
	Uplink      string
	UplinkSpeed int
	Load        float64

	Radios map[string]int
}
