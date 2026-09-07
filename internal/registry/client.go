package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// BaseURL normalizes hub address.
// Accepts: host:port | http://host:port | http://host/path | https://host/path
func BaseURL(hub string) string {
	hub = strings.TrimSpace(hub)
	if hub == "" {
		return ""
	}
	if !strings.HasPrefix(hub, "http://") && !strings.HasPrefix(hub, "https://") {
		hub = "http://" + hub
	}
	return strings.TrimRight(hub, "/")
}

func apiURL(hub, path string) string {
	return BaseURL(hub) + path
}

// Client registers this machine with a remote registry hub.
type Client struct {
	hub    string // full base URL
	self   registerReq
	stop   chan struct{}
	httpC  *http.Client
}

func NewClient(hub string, name, ip string, httpPort int, pinSet bool, version string) *Client {
	return NewClientMulti(hub, name, []string{ip}, httpPort, pinSet, version)
}

func NewClientMulti(hub string, name string, ips []string, httpPort int, pinSet bool, version string) *Client {
	primary := ""
	if len(ips) > 0 {
		primary = ips[0]
	}
	id := primary
	if id == "" {
		id = "self"
	}
	return &Client{
		hub: BaseURL(hub),
		self: registerReq{
			ID:       id + ":" + itoa(httpPort),
			Name:     name,
			IP:       primary,
			IPs:      ips,
			HTTPPort: httpPort,
			PINSet:   pinSet,
			Version:  version,
		},
		stop:  make(chan struct{}),
		httpC: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) SetPINSet(v bool) {
	c.self.PINSet = v
}

func (c *Client) SetName(n string) {
	c.self.Name = n
}

func (c *Client) post(path string, body interface{}) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := apiURL(c.hub, path)
	resp, err := c.httpC.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s -> %d", path, resp.StatusCode)
	}
	return nil
}

func (c *Client) Register() error {
	return c.post("/api/register", c.self)
}

func (c *Client) Heartbeat() error {
	return c.post("/api/heartbeat", c.self)
}

func (c *Client) Unregister() {
	_ = c.post("/api/unregister", c.self)
}

func (c *Client) Start() {
	go func() {
		_ = c.Register()
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-c.stop:
				c.Unregister()
				return
			case <-t.C:
				if err := c.Heartbeat(); err != nil {
					_ = c.Register()
				}
			}
		}
	}()
}

func (c *Client) Stop() {
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
}

// FetchDevices pulls the online list from a hub (host:port or full URL with path).
func FetchDevices(hub string) ([]Device, error) {
	hc := &http.Client{Timeout: 5 * time.Second}
	resp, err := hc.Get(apiURL(hub, "/api/devices"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("devices -> %d", resp.StatusCode)
	}
	var out struct {
		Devices []Device `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Devices, nil
}
