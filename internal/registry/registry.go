package registry

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	defaultTTL = 20 * time.Second
	sweepEvery = 5 * time.Second
)

// Device is a registered control endpoint.
type Device struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	IP       string    `json:"ip"`   // primary / preferred
	IPs      []string  `json:"ips"`  // all NICs
	HTTPPort int       `json:"http_port"`
	PINSet   bool      `json:"pin_set"`
	Version  string    `json:"version"`
	Seen     time.Time `json:"seen"`
	Addr     string    `json:"addr"`   // ip:http_port (preferred)
	Addrs    []string  `json:"addrs"`  // all ip:port candidates
}

type registerReq struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	IP       string   `json:"ip"`
	IPs      []string `json:"ips"`
	HTTPPort int      `json:"http_port"`
	PINSet   bool     `json:"pin_set"`
	Version  string   `json:"version"`
}

// Server is the LAN directory / registry on its own port.
type Server struct {
	mu      sync.Mutex
	devices map[string]*Device // key: id
	port    int
	srv     *http.Server
}

func New(port int) *Server {
	if port <= 0 {
		port = 8760
	}
	return &Server{
		devices: map[string]*Device{},
		port:    port,
	}
}

func (s *Server) Port() int { return s.port }

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/register", s.handleRegister)
	mux.HandleFunc("/api/heartbeat", s.handleHeartbeat)
	mux.HandleFunc("/api/unregister", s.handleUnregister)
	mux.HandleFunc("/api/devices", s.handleDevices)

	ln, err := net.Listen("tcp", addr(s.port))
	if err != nil {
		return err
	}
	if ta, ok := ln.Addr().(*net.TCPAddr); ok {
		s.port = ta.Port
	}
	s.srv = &http.Server{Handler: mux}
	log.Printf("registry listening on :%d", s.port)
	go s.sweepLoop()
	return s.srv.Serve(ln)
}

func addr(port int) string { return ":" + itoa(port) }

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [12]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}

func (s *Server) sweepLoop() {
	t := time.NewTicker(sweepEvery)
	defer t.Stop()
	for range t.C {
		cut := time.Now().Add(-defaultTTL)
		s.mu.Lock()
		for k, d := range s.devices {
			if d.Seen.Before(cut) {
				delete(s.devices, k)
				log.Printf("registry: drop offline %s (%s)", d.Name, d.Addr)
			}
		}
		s.mu.Unlock()
	}
}

func (s *Server) upsert(req registerReq, sourceIP string) Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := req.ID
	if id == "" {
		id = req.IP + ":" + itoa(req.HTTPPort)
	}

	ips := make([]string, 0, len(req.IPs)+2)
	seen := map[string]bool{}
	addIP := func(ip string) {
		if ip == "" || seen[ip] {
			return
		}
		seen[ip] = true
		ips = append(ips, ip)
	}
	// Prefer the IP that actually reached this hub (same L2/L3 as controller path)
	addIP(sourceIP)
	addIP(req.IP)
	for _, ip := range req.IPs {
		addIP(ip)
	}

	addrs := make([]string, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, net.JoinHostPort(ip, itoa(req.HTTPPort)))
	}

	preferred := ""
	if len(ips) > 0 {
		preferred = ips[0]
	}
	addr := ""
	if len(addrs) > 0 {
		addr = addrs[0]
	}

	d := &Device{
		ID:       id,
		Name:     req.Name,
		IP:       preferred,
		IPs:      ips,
		HTTPPort: req.HTTPPort,
		PINSet:   req.PINSet,
		Version:  req.Version,
		Seen:     time.Now(),
		Addr:     addr,
		Addrs:    addrs,
	}
	s.devices[id] = d
	return *d
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.HTTPPort <= 0 {
		http.Error(w, "bad request", 400)
		return
	}
	if req.IP == "" {
		req.IP = clientIP(r)
	}
	d := s.upsert(req, clientIP(r))
	log.Printf("registry: + %s %s addrs=%v", d.Name, d.Addr, d.Addrs)
	writeJSON(w, map[string]interface{}{"ok": true, "device": d})
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	if req.IP == "" {
		req.IP = clientIP(r)
	}
	d := s.upsert(req, clientIP(r))
	writeJSON(w, map[string]interface{}{"ok": true, "device": d})
}

func (s *Server) handleUnregister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	id := req.ID
	if id == "" {
		id = req.IP + ":" + itoa(req.HTTPPort)
	}
	s.mu.Lock()
	delete(s.devices, id)
	s.mu.Unlock()
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	out := make([]Device, 0, len(s.devices))
	cut := time.Now().Add(-defaultTTL)
	for _, d := range s.devices {
		if d.Seen.Before(cut) {
			continue
		}
		out = append(out, *d)
	}
	s.mu.Unlock()
	writeJSON(w, map[string]interface{}{"devices": out})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
