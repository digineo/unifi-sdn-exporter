package exporter

import (
	"context"
	"log"

	"github.com/digineo/unifi-sdn-exporter/unifi"
	"github.com/prometheus/client_golang/prometheus"
)

type unifiCollector struct {
	client unifi.Client
	ctx    context.Context
	site   string
}

var _ prometheus.Collector = (*unifiCollector)(nil)

var (
	ctrlUp = ctrlDesc("up", "indicator whether controller is reachable", "version")

	siteWifiUtil         = siteDesc("wifi_utilization", "average Wifi utilization", "band")
	siteWifiClientsScore = siteDesc("wifi_client_score", "average client score") // 0-100?
	siteWifiClientsCount = siteDesc("wifi_clients_count", "number of clients by rating", "rating")

	devLabel    = []string{"mac"}
	devStatus   = deviceDesc("status", "current device status", "desc", "model_id", "model", "firmware")
	devUptime   = deviceDesc("uptime", "uptime of device in seconds")
	devLoad     = deviceDesc("load", "current system load of endpoint")
	devClients  = deviceDesc("clients", "number of connected WLAN clients", "band")
	devUplink   = deviceDesc("uplink", "uplink type and speed", "type")
	devLastSeen = deviceDesc("last_seen", "Unix timestamp when the device was last seen")
)

func (uc *unifiCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- ctrlUp

	ch <- siteWifiUtil
	ch <- siteWifiClientsScore
	ch <- siteWifiClientsCount

	ch <- devStatus
	ch <- devUptime
	ch <- devLoad
	ch <- devClients
	ch <- devUplink
	ch <- devLastSeen
}

func (uc *unifiCollector) Collect(ch chan<- prometheus.Metric) {
	const C, G = prometheus.CounterValue, prometheus.GaugeValue

	metric := func(desc *prometheus.Desc, typ prometheus.ValueType, v float64, label ...string) {
		ch <- prometheus.MustNewConstMetric(desc, typ, v, label...)
	}

	m, err := uc.client.Metrics(uc.ctx, uc.site)
	if err != nil {
		metric(ctrlUp, G, 0, "")
		log.Println("fetching failed:", err)
		return
	}

	metric(ctrlUp, G, 1, m.ControllerVersion)
	metric(siteWifiUtil, G, m.AvgWifiUtilization24, "2.4")
	metric(siteWifiUtil, G, m.AvgWifiUtilization50, "5")
	metric(siteWifiClientsScore, G, m.AvgWifiScore)
	metric(siteWifiClientsCount, G, float64(m.ClientsPoorScore), "poor")
	metric(siteWifiClientsCount, G, float64(m.ClientsFairScore), "fair")
	metric(siteWifiClientsCount, G, float64(m.ClientsGoodScore), "good")

	for _, d := range m.Devices {
		metric(devStatus, G, float64(d.Status), d.MAC, d.StatusHuman, d.Model, d.ModelHuman, d.Firmware)

		if !d.LastSeen.IsZero() {
			metric(devLastSeen, G, float64(d.LastSeen.Unix()), d.MAC)
		}
		if d.Uptime != nil {
			metric(devUptime, G, d.Uptime.Seconds(), d.MAC)
		}
		if d.Load != nil {
			metric(devLoad, G, *d.Load, d.MAC)
		}
		if d.Uplink != nil {
			metric(devUplink, G, float64(*d.UplinkSpeed), d.MAC, *d.Uplink)
		}

		for band, clients := range d.Radios {
			metric(devClients, G, float64(clients), d.MAC, band)
		}
	}
}

func ctrlDesc(name, help string, extraLabel ...string) *prometheus.Desc {
	fqdn := prometheus.BuildFQName("unifi_sdn", "controller", name)
	return prometheus.NewDesc(fqdn, help, extraLabel, nil)
}

func siteDesc(name, help string, extraLabel ...string) *prometheus.Desc {
	fqdn := prometheus.BuildFQName("unifi_sdn", "site", name)
	return prometheus.NewDesc(fqdn, help, extraLabel, nil)
}

func deviceDesc(name, help string, extraLabel ...string) *prometheus.Desc {
	fqdn := prometheus.BuildFQName("unifi_sdn", "device", name)
	return prometheus.NewDesc(fqdn, help, append(devLabel, extraLabel...), nil)
}
