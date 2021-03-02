package unifi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Controller struct {
	Alias    string
	Username string
	Password string
	URL      string
	Insecure bool // skip server certificate check if scheme is https, but the certificate is self-signed

	init     bool
	client   *http.Client
	endpoint *url.URL

	sitesCache        []sitesResponse // maps ident to site
	sitesCacheExpires time.Time       // marks expiry for cache
	sitesCacheMu      sync.RWMutex    // protects sitesCache and expiry
}

type Client interface {
	Metrics(ctx context.Context, siteDesc string) (*Metrics, error)
	Get(ctx context.Context, path string, res interface{}) error
	Sites(ctx context.Context) ([]Site, error)
}

var _ Client = (*Controller)(nil)

// Verbose increases verbosity.
var Verbose bool

func vlogf(format string, v ...interface{}) {
	if Verbose {
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

func vlog(s string) {
	if Verbose {
		log.Output(2, s)
	}
}

const (
	sessionCookieName = "unifises"
	siteCacheTTL      = 5 * time.Minute
)

// NewClient creates a new Client instance.
func NewClient(ctrl *Controller) (Client, error) {
	if ctrl.init {
		return ctrl, nil
	}

	if ctrl.Username == "" || ctrl.Password == "" {
		return nil, ErrMissingCredentials
	}

	endpoint, err := url.Parse(ctrl.URL)
	if err != nil {
		return nil, &ErrInvalidEndpoint{err}
	}

	ctrl.endpoint = endpoint
	ctrl.client = &http.Client{
		Timeout: 10 * time.Second,
	}
	ctrl.client.Jar, _ = cookiejar.New(nil) // error is always nil

	if endpoint.Scheme == "https" {
		ctrl.client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: ctrl.Insecure,
			},
		}
	}
	ctrl.init = true

	return ctrl, nil
}

func (c *Controller) login(ctx context.Context) error {
	c.client.Jar.SetCookies(c.endpoint, []*http.Cookie{{
		Name:   sessionCookieName,
		MaxAge: -1,
	}})

	req := loginRequest{
		Username: c.Username,
		Password: c.Password,
		Remember: true,
		Strict:   false,
	}
	err := c.apiRequest(ctx, http.MethodPost, loginPath, &req, nil)
	if err != nil {
		return err
	}

	return nil
}

// apiRequest sends an API request to the controller. The path is constructed
// from c.base + "/api/" + path. The request parameter, if not nil, will be
// JSON encoded, and the JSON response is decoded into the response parameter.
//
//	req, res := requestType{...}, responseType{...}
//	err := c.apiRequest(ctx, "POST", "node/status", &req, &res)
func (c *Controller) apiRequest(ctx context.Context, method, path string, request, response interface{}) error {
	url := fmt.Sprintf("%s://%s/%s", c.endpoint.Scheme, c.endpoint.Host, strings.TrimPrefix(path, "/"))
	vlogf("%s %s", method, url)

	var body io.Reader
	if request != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(request); err != nil {
			return fmt.Errorf("encoding body failed: %w", err)
		}
		body = &buf
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("cannot construct request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if request != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		data, _ := ioutil.ReadAll(res.Body)

		return &ErrUnexpectedStatus{
			Method: method,
			URL:    url,
			Status: res.StatusCode,
			Body:   data,
		}
	}

	jsonData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// parse response
	var meta metaResponse
	if err = json.Unmarshal(jsonData, &meta); err != nil {
		return fmt.Errorf("decoding meta response failed: %w", err)
	}
	if meta.Meta.RC != "ok" {
		return ErrRequestFailed(meta.Meta.Message)
	}
	if meta.Data == nil {
		return &genericError{msg: "missing response payload"}
	}

	// caller wants metaResponse
	if m, ok := response.(*metaResponse); ok {
		*m = meta
		return nil
	}

	err = json.Unmarshal(*meta.Data, &response)
	if err != nil {
		vlog(string(jsonData))
		vlog(err.Error())
		return fmt.Errorf("decoding response failed: %w", err)
	}

	return nil
}

func (c *Controller) Get(ctx context.Context, path string, res interface{}) error {
	retried := false

retry:
	errStatus := &ErrUnexpectedStatus{}
	err := c.apiRequest(ctx, http.MethodGet, path, nil, res)

	if errors.As(err, &errStatus) && errStatus.Status == http.StatusUnauthorized && !retried {
		vlog("unauthorized, logging in")
		err = c.login(ctx)
		if err == nil {
			retried = true
			goto retry
		}
	}
	if err != nil {
		vlogf("request failed: %v", err)
	}
	return err
}

type Site struct {
	Name, Desc string
}

func (c *Controller) Sites(ctx context.Context) (sites []Site, err error) {
	if c.sitesCacheExpires.Before(time.Now().Add(-siteCacheTTL)) {
		if err := c.updateSiteCache(ctx); err != nil {
			return nil, err
		}
	}

	c.sitesCacheMu.RLock()
	defer c.sitesCacheMu.RUnlock()

	for _, s := range c.sitesCache {
		sites = append(sites, Site{s.Name, s.Desc})
	}
	return
}

func (c *Controller) fetchSite(ctx context.Context, ident string) (*sitesResponse, error) {
	if c.sitesCacheExpires.Before(time.Now().Add(-siteCacheTTL)) {
		if err := c.updateSiteCache(ctx); err != nil {
			return nil, err
		}
	}

	c.sitesCacheMu.RLock()
	defer c.sitesCacheMu.RUnlock()

	for _, s := range c.sitesCache {
		if s.Name == ident || s.Desc == ident {
			return &s, nil
		}
	}

	return nil, nil
}

func (c *Controller) updateSiteCache(ctx context.Context) error {
	c.sitesCacheMu.Lock()
	defer c.sitesCacheMu.Unlock()

	var sites []sitesResponse
	if err := c.Get(ctx, sitesPath, &sites); err != nil {
		return err
	}

	c.sitesCache = sites
	c.sitesCacheExpires = time.Now().Add(siteCacheTTL)
	return nil
}

func (c *Controller) Metrics(ctx context.Context, siteDesc string) (*Metrics, error) {
	site, err := c.fetchSite(ctx, siteDesc)
	if err != nil {
		return nil, err
	}

	sitepath := func(p string) string {
		return strings.Replace(p, "{siteName}", site.Name, 1)
	}

	vlog("fetching status")
	status := metaResponse{}
	if err := c.Get(ctx, statusPath, &status); err != nil {
		return nil, err
	}

	vlog("fetching health info")
	health := []siteHealthResponse{}
	if err := c.Get(ctx, sitepath(siteHealthPath), &health); err != nil {
		return nil, err
	}
	if len(health) != 1 {
		return nil, &genericError{"unexpected result length"}
	}

	vlog("fetching device statistics")
	devices := []siteDeviceResponse{}
	if err := c.Get(ctx, sitepath(siteDevicesPath), &devices); err != nil {
		return nil, err
	}

	util := health[0].AvgWifiUtilization
	score := health[0].WifiScore
	m := &Metrics{
		ControllerVersion:    status.Meta.ServerVersion,
		AvgWifiUtilization24: util.Band24,
		AvgWifiUtilization50: util.Band5,
		AvgWifiScore:         score.ClientScoreAvg,
		ClientsPoorScore:     score.PoorClients,
		ClientsFairScore:     score.FairClients,
		ClientsGoodScore:     score.TotalClients - (score.PoorClients + score.FairClients),
	}

	for _, d := range devices {
		if !d.Adopted {
			continue // unadopted devices show up in *every* site
		}

		lastSeen := time.Unix(int64(d.LastSeenUnix), 0)
		var load float64
		if l := d.Sys.Load1; l != "" {
			if l[0] == '"' && l[len(l)-1] == '"' {
				l = l[1 : len(l)-1]
			}
			load64, err := strconv.ParseFloat(l, 64)
			if err != nil {
				load = -1
			} else {
				load = load64
			}
		}

		dm := DeviceMetrics{
			MAC:         d.MAC,
			Firmware:    d.Version,
			Model:       d.Model,
			ModelHuman:  d.ModelHuman(),
			LTS:         d.LTS,
			EOL:         d.EOL,
			Status:      int(d.State),
			StatusHuman: d.State.String(),
			Uptime:      time.Duration(d.Uptime) * time.Second, //nolint:durationcheck
			LastSeen:    &lastSeen,
			Uplink:      d.UplinkDescription(),
			UplinkSpeed: d.UplinkSpeed(),
			Load:        load,
			Radios:      make(map[string]int),
		}

		for _, vap := range d.VAP {
			dm.Radios[Band(vap.Radio)] += vap.Clients
		}

		m.Devices = append(m.Devices, dm)
	}

	return m, nil
}
