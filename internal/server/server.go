package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"lan-remote/internal/capture"
	"lan-remote/internal/inject"

	"github.com/gorilla/websocket"
)

type Config struct {
	Addr    string // e.g. ":8765"
	PIN     string
	Quality int    // JPEG quality 1-100
	FPS     int
	// Peers is optional; used by /api/peers for the desktop UI.
	Peers func() interface{}
	// OnPINChanged is called after PIN is updated via admin API.
	OnPINChanged func(pin string)
	// Extra fields exposed on /api/status (hub, role, version…).
	Extra map[string]string
	// OnSetHub validates and stores the registry address (client mode).
	// Return error to reject. Empty hub clears.
	OnSetHub func(hub string) error
}

type Server struct {
	cfg  Config
	cap  capture.Capturer
	inj  inject.Controller
	upg  websocket.Upgrader
	mu   sync.Mutex
	stop chan struct{}

	activeConns int
	port        int
	hostname    string
}

func New(cfg Config) *Server {
	if cfg.Quality <= 0 {
		cfg.Quality = 70
	}
	if cfg.Quality > 100 {
		cfg.Quality = 100
	}
	if cfg.FPS <= 0 {
		cfg.FPS = 15
	}
	if cfg.FPS > 240 {
		cfg.FPS = 240
	}
	hn, _ := os.Hostname()
	return &Server{
		cfg:      cfg,
		cap:      capture.New(),
		inj:      inject.New(),
		upg:      websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		stop:     make(chan struct{}),
		hostname: hn,
	}
}

// GeneratePIN returns a 6-digit numeric PIN.
func GeneratePIN() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	pin := ""
	for i := 0; i < 6; i++ {
		pin += strconv.Itoa(int(b[i])%10)
	}
	return pin
}

func (s *Server) LANAddrs() []string {
	var out []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			// skip link-local 169.254/16
			if ip[0] == 169 && ip[1] == 254 {
				continue
			}
			out = append(out, ip.String())
		}
	}
	return out
}

func (s *Server) Addr() string { return s.cfg.Addr }

func (s *Server) PIN() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.PIN
}

func (s *Server) SetPIN(pin string) {
	s.mu.Lock()
	s.cfg.PIN = pin
	s.mu.Unlock()
}

func (s *Server) Port() int      { return s.port }
func (s *Server) Hostname() string { return s.hostname }

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/pin", s.handleSetPIN)
	mux.HandleFunc("/api/peers", s.handlePeers)
	mux.HandleFunc("/api/hub", s.handleSetHub)
	mux.HandleFunc("/api/file", s.handleUpload)
	mux.HandleFunc("/api/download", s.handleDownload)

	srv := &http.Server{Addr: s.cfg.Addr, Handler: mux}

	// resolve bound port
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	if ta, ok := ln.Addr().(*net.TCPAddr); ok {
		s.port = ta.Port
	}
	log.Printf("listening on %s  PIN=%s", ln.Addr().String(), s.PIN())
	for _, ip := range s.LANAddrs() {
		log.Printf("  http://%s:%d", ip, s.port)
	}
	return srv.Serve(ln)
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	out := map[string]interface{}{
		"ok":       true,
		"name":     s.hostname,
		"port":     s.port,
		"ips":      s.LANAddrs(),
		"pin_set":  s.PIN() != "",
		"clients":  s.activeConns,
		"quality":  s.cfg.Quality,
		"fps":      s.cfg.FPS,
		"platform": "server",
	}
	for k, v := range s.cfg.Extra {
		out[k] = v
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleSetPIN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if !isLoopbackRequest(r) {
		http.Error(w, "PIN 只能在被控机本机修改（请在该电脑上打开应用窗口）", 403)
		return
	}
	var body struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PIN == "" {
		http.Error(w, "bad request", 400)
		return
	}
	if len(body.PIN) < 4 || len(body.PIN) > 12 {
		http.Error(w, "pin length 4-12", 400)
		return
	}
	s.SetPIN(body.PIN)
	if s.cfg.OnPINChanged != nil {
		s.cfg.OnPINChanged(body.PIN)
	}
	log.Printf("PIN changed (localhost)")
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.cfg.Peers == nil {
		_, _ = w.Write([]byte(`{"peers":[]}`))
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"peers": s.cfg.Peers()})
}

func (s *Server) handleSetHub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if s.cfg.OnSetHub == nil {
		http.Error(w, "hub not supported", 400)
		return
	}
	var body struct {
		Hub string `json:"hub"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	if err := s.cfg.OnSetHub(body.Hub); err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	s.mu.Lock()
	if s.cfg.Extra == nil {
		s.cfg.Extra = map[string]string{}
	}
	s.cfg.Extra["hub"] = body.Hub
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "hub": body.Hub})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

type wsMsg struct {
	Type    string  `json:"type"`
	X       float64 `json:"x,omitempty"`
	Y       float64 `json:"y,omitempty"`
	Button  string  `json:"button,omitempty"`
	Down    bool    `json:"down,omitempty"`
	Key     string  `json:"key,omitempty"`
	Text    string  `json:"text,omitempty"`
	DX      float64 `json:"dx,omitempty"`
	DY      float64 `json:"dy,omitempty"`
	PIN     string  `json:"pin,omitempty"`
	Quality int     `json:"quality,omitempty"`
	FPS     int     `json:"fps,omitempty"`
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upg.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	// Auth
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return
	}
	var auth wsMsg
	if err := json.Unmarshal(raw, &auth); err != nil || auth.PIN != s.PIN() || s.PIN() == "" {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"auth","ok":false}`))
		return
	}
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"auth","ok":true}`))
	conn.SetReadDeadline(time.Time{})

	s.mu.Lock()
	s.activeConns++
	n := s.activeConns
	s.mu.Unlock()
	log.Printf("client connected (%d active)", n)
	defer func() {
		s.mu.Lock()
		s.activeConns--
		s.mu.Unlock()
		log.Printf("client disconnected")
	}()

	// Only stream to the first viewer; control is always accepted.
	// (simple: stream to every authenticated conn — LAN is fine)
	done := make(chan struct{})
	go s.streamLoop(conn, done)
	s.readLoop(conn, done)
	close(done)
}

func (s *Server) readLoop(conn *websocket.Conn, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var m wsMsg
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		s.handleInput(&m)
	}
}

func (s *Server) handleInput(m *wsMsg) {
	switch m.Type {
	case "move":
		s.inj.MouseMove(int(m.X), int(m.Y))
	case "button":
		log.Printf("input button=%s down=%v", m.Button, m.Down)
		s.inj.MouseButton(m.Button, m.Down)
	case "scroll":
		s.inj.MouseScroll(int(m.DX), int(m.DY))
	case "key":
		log.Printf("input key=%s down=%v", m.Key, m.Down)
		s.inj.Key(m.Key, m.Down)
	case "text":
		if m.Text != "" {
			log.Printf("input text len=%d", len(m.Text))
			s.inj.Text(m.Text)
		}
	case "settings":
		if m.Quality >= 1 && m.Quality <= 100 {
			s.cfg.Quality = m.Quality
		}
		if m.FPS >= 1 && m.FPS <= 240 {
			s.cfg.FPS = m.FPS
		}
	}
}

func (s *Server) streamLoop(conn *websocket.Conn, done chan struct{}) {
	interval := time.Second / time.Duration(s.cfg.FPS)
	t := time.NewTicker(interval)
	defer t.Stop()
	lastQ := s.cfg.Quality
	lastF := s.cfg.FPS
	for {
		select {
		case <-done:
			return
		case <-t.C:
			// apply settings changes without restarting ticker if fps unchanged
			if s.cfg.FPS != lastF {
				lastF = s.cfg.FPS
				interval = time.Second / time.Duration(lastF)
				t.Reset(interval)
			}
			lastQ = s.cfg.Quality
			img, err := s.cap.Capture()
			if err != nil {
				log.Println("capture:", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			buf := make([]byte, 0, 256*1024)
			w := &sliceWriter{b: &buf}
			if err := jpeg.Encode(w, img, &jpeg.Options{Quality: lastQ}); err != nil {
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.BinaryMessage, buf); err != nil {
				return
			}
		}
	}
}

type sliceWriter struct{ b *[]byte }

func (w *sliceWriter) Write(p []byte) (int, error) {
	*w.b = append(*w.b, p...)
	return len(p), nil
}

// indexHTML is embedded (see web.go).
var _ = fmt.Sprintf
