package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client registers this machine with a remote registry hub.
type Client struct {
	hub    string // host:port
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
		hub: hub,
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
		httpC: &http.Client{Timeout: 3 * time.Second},
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
	url := "http://" + c.hub + path
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
					// re-register if hub restarted
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

// FetchDevices pulls the online list from a hub.
func FetchDevices(hub string) ([]Device, error) {
	hc := &http.Client{Timeout: 3 * time.Second}
	resp, err := hc.Get("http://" + hub + "/api/devices")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Devices []Device `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Devices, nil
}
