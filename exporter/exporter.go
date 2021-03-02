package exporter

import (
	_ "embed" //nolint:golint
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"text/template"

	"github.com/digineo/unifi-sdn-exporter/unifi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (cfg *Config) Start(listenAddress string) {
	http.Handle("/metrics", cfg.targetMiddleware(cfg.metricsHandler))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI != "/" {
			http.NotFound(w, r)
			return
		}

		vars := make(map[string][]unifi.Site)

		for _, ctrl := range cfg.Controllers {
			client, err := unifi.NewClient(ctrl)
			if err != nil {
				http.Error(w, fmt.Sprintf("error creating client for controller %s: %v", ctrl.Alias, err), http.StatusInternalServerError)
				return
			}

			s, err := client.Sites(r.Context())
			if err != nil {
				http.Error(w, fmt.Sprintf("error fetching sites for controller %s: %v", ctrl.Alias, err), http.StatusInternalServerError)
				return
			}

			sort.Slice(s, func(i, j int) bool {
				return strings.Compare(s[i].Desc, s[j].Desc) < 0
			})

			vars[ctrl.Alias] = s
		}

		tmpl.Execute(w, vars)
	})

	log.Printf("Starting exporter on http://%s/", listenAddress)
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}

type targetHandler func(unifi.Client, string, http.ResponseWriter, *http.Request)

func (cfg *Config) targetMiddleware(next targetHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("target")
		if target == "" {
			http.Error(w, "target parameter missing", http.StatusBadRequest)
			return
		}

		site := r.URL.Query().Get("site")
		if site == "" {
			http.Error(w, "site parameter missing", http.StatusBadRequest)
			return
		}

		client, err := cfg.getClient(target)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		if client == nil {
			http.Error(w, "configuration not found", http.StatusNotFound)
			return
		}

		next(client, site, w, r)
	})
}

func (cfg *Config) metricsHandler(client unifi.Client, site string, w http.ResponseWriter, r *http.Request) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(&unifiCollector{
		client: client,
		ctx:    r.Context(),
		site:   site,
	})
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

//go:embed index.tpl.html
var rawTmpl string

var tmpl = template.Must(template.New("index").Option("missingkey=error").Parse(rawTmpl))
