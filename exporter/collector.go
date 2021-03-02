package exporter

import (
	"context"
	"log"

	"git.digineo.de/digineo/unifi-sdn-exporter/unifi"
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
	siteWifiClientsPoor  = siteDesc("wifi_clients_poor_count", "number of clients with poor score")
	siteWifiClientsFair  = siteDesc("wifi_clients_fair_count", "number of clients with fair score")
	siteWifiClientsGood  = siteDesc("wifi_clients_good_count", "number of clients with good score")

	devLabel   = []string{"mac"}
	devStatus  = deviceDesc("status", "current device status", "desc", "model_id", "model", "firmware")
	devUptime  = deviceDesc("uptime", "uptime of device in seconds")
	devLoad    = deviceDesc("load", "current system load of endpoint")
	devClients = deviceDesc("clients", "number of connected WLAN clients", "band")
)

func (uc *unifiCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- ctrlUp

	ch <- siteWifiUtil
	ch <- siteWifiClientsScore
	ch <- siteWifiClientsPoor
	ch <- siteWifiClientsFair
	ch <- siteWifiClientsGood

	ch <- devStatus
	ch <- devUptime
	ch <- devLoad
	ch <- devClients
}

func (uc *unifiCollector) Collect(ch chan<- prometheus.Metric) {
	const C, G = prometheus.CounterValue, prometheus.GaugeValue

	metric := func(desc *prometheus.Desc, typ prometheus.ValueType, v float64, label ...string) {
		ch <- prometheus.MustNewConstMetric(desc, typ, v, label...)
	}

	m, err := uc.client.Metrics(uc.ctx, uc.site)
	metric(ctrlUp, G, boolToFloat(err == nil), m.ControllerVersion)
	if err != nil {
		log.Println("fetching failed:", err)
		return
	}

	metric(siteWifiUtil, G, m.AvgWifiUtilization24, "2.4")
	metric(siteWifiUtil, G, m.AvgWifiUtilization50, "5")
	metric(siteWifiClientsScore, G, m.AvgWifiScore)
	metric(siteWifiClientsPoor, G, float64(m.ClientsPoorScore))
	metric(siteWifiClientsFair, G, float64(m.ClientsFairScore))
	metric(siteWifiClientsGood, G, float64(m.ClientsGoodScore))

	for _, d := range m.Devices {
		metric(devStatus, G, float64(d.Status), d.MAC, d.StatusHuman, d.Model, d.ModelHuman, d.Firmware)

		if d.Uptime == 0 {
			metric(devUptime, G, 0, d.MAC)
			continue
		}
		metric(devUptime, G, d.Uptime.Seconds(), d.MAC)
		metric(devLoad, G, d.Load, d.MAC)

		for band, clients := range d.Radios {
			metric(devClients, G, float64(clients), d.MAC, band)
		}
	}
}

func boolToFloat(val bool) float64 {
	if val {
		return 1
	}

	return 0
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
