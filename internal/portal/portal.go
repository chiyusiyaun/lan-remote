package portal

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"lan-remote/internal/registry"

	"github.com/gorilla/websocket"
)

//go:embed index.html
var indexHTML string

// Config for the unified portal (usually on the server host :8765).
type Config struct {
	Addr     string // e.g. ":8765"
	Registry string // host:port of registry, e.g. 127.0.0.1:8760
	Version  string
}

type Server struct {
	cfg Config
	upg websocket.Upgrader
}

func New(cfg Config) *Server {
	if cfg.Version == "" {
		cfg.Version = "1.2.1"
	}
	return &Server{
		cfg: cfg,
		upg: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	log.Printf("portal listening on %s", ln.Addr().String())
	return http.Serve(ln, mux)
}

// RegisterRoutes mounts portal UI + relay APIs (does not own "/" if shared).
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/peers", s.handlePeers)
	mux.HandleFunc("/api/file", s.handleFileRelay)
	mux.HandleFunc("/api/file/", s.handleFileRelay)
	mux.HandleFunc("/api/files", s.handleFilesRelay)
	mux.HandleFunc("/api/files/", s.handleFilesRelay)
	mux.HandleFunc("/api/download", s.handleDownloadRelay)
	mux.HandleFunc("/api/download/", s.handleDownloadRelay)
	mux.HandleFunc("/api/mkdir", s.handleMkdirRelay)
	mux.HandleFunc("/api/mkdir/", s.handleMkdirRelay)
	mux.HandleFunc("/proxy", s.handleProxy)
	mux.HandleFunc("/proxy/", s.handleProxy)
}

// extractTarget reads target from ?target= or path suffix (host:port).
func extractTarget(r *http.Request, prefix string) string {
	t := strings.TrimSpace(r.URL.Query().Get("target"))
	if t != "" {
		return normTarget(t)
	}
	p := strings.TrimPrefix(r.URL.Path, prefix)
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	// /api/file/host:port  or  /api/file/host/port
	if strings.Count(p, "/") == 0 && strings.Contains(p, ":") {
		return normTarget(p)
	}
	return normTarget(strings.Replace(p, "/", ":", 1))
}

func normTarget(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return ""
	}
	if !strings.Contains(t, ":") {
		t += ":8765"
	}
	return t
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

// ServeIndex returns the portal UI handler for mux "/".
func (s *Server) ServeIndex() http.HandlerFunc { return s.handleIndex }

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"app":     "portal",
		"role":    "portal",
		"hub":     s.cfg.Registry,
		"proxy":   true,
		"version": s.cfg.Version,
	})
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	devs, err := registry.FetchDevices(s.cfg.Registry)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	// shape compatible with client UI (/api/peers -> {peers:[...]})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"peers": devs})
}

// handleProxy relays a browser WebSocket to a target client control port.
// handleProxy relays a browser WebSocket to a target client control port.
//
// Supported:
//
//	ws://server/proxy?target=192.168.1.20:8765
//	ws://server/proxy/192.168.1.20:8765
//	ws://server/proxy/192.168.1.20/8765
//
// Path form works behind gateways that drop query strings.
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	if target == "" {
		// path: /proxy/host:port  or  /proxy/host/port
		p := strings.TrimPrefix(r.URL.Path, "/proxy")
		p = strings.Trim(p, "/")
		if p != "" {
			target = strings.Replace(p, "/", ":", 1) // host/port -> host:port
			// if already host:port keep as-is
			if strings.Count(p, "/") == 0 && strings.Contains(p, ":") {
				target = p
			}
		}
	}
	target = strings.TrimSpace(target)
	if target == "" {
		http.Error(w, "target required (use /proxy/host:port or ?target=host:port)", 400)
		return
	}
	if !strings.Contains(target, ":") {
		target += ":8765"
	}
	// refuse loopback targets that would only work on the server itself
	if host, _, err := net.SplitHostPort(target); err == nil {
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			http.Error(w, "loopback target not allowed", 400)
			return
		}
	}

	clientConn, err := s.upg.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()

	upURL := url.URL{Scheme: "ws", Host: target, Path: "/ws"}
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	upConn, _, err := dialer.Dial(upURL.String(), nil)
	if err != nil {
		log.Println("proxy dial", target, err)
		_ = clientConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"auth","ok":false,"error":"upstream"}`))
		return
	}
	defer upConn.Close()
	log.Println("proxy", r.RemoteAddr, "->", target)

	errc := make(chan error, 2)
	go func() {
		for {
			mt, msg, err := clientConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			_ = upConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := upConn.WriteMessage(mt, msg); err != nil {
				errc <- err
				return
			}
		}
	}()
	go func() {
		for {
			mt, msg, err := upConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			_ = clientConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := clientConn.WriteMessage(mt, msg); err != nil {
				errc <- err
				return
			}
		}
	}()
	<-errc
}

// handleFileRelay proxies POST /api/file[?target=|/host:port] to the client.
func (s *Server) handleFileRelay(w http.ResponseWriter, r *http.Request) {
	target := extractTarget(r, "/api/file")
	if target == "" {
		http.Error(w, "target required", 400)
		return
	}
	pin := r.Header.Get("X-LR-Pin")
	q := r.URL.Query()
	q.Del("target")
	upURL := "http://" + target + "/api/file"
	if enc := q.Encode(); enc != "" {
		upURL += "?" + enc
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Set("X-LR-Pin", pin)
	req.ContentLength = r.ContentLength
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "upstream: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) relayGET(w http.ResponseWriter, r *http.Request, prefix, upstreamPath string) {
	target := extractTarget(r, prefix)
	if target == "" {
		http.Error(w, "target required", 400)
		return
	}
	if !strings.Contains(target, ":") {
		target += ":8765"
	}
	q := r.URL.Query()
	q.Del("target")
	u := "http://" + target + upstreamPath
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if pin := r.Header.Get("X-LR-Pin"); pin != "" {
		req.Header.Set("X-LR-Pin", pin)
	}
	if pin := q.Get("pin"); pin != "" {
		req.Header.Set("X-LR-Pin", pin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "upstream: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleFilesRelay(w http.ResponseWriter, r *http.Request) {
	s.relayGET(w, r, "/api/files", "/api/files")
}

func (s *Server) handleDownloadRelay(w http.ResponseWriter, r *http.Request) {
	s.relayGET(w, r, "/api/download", "/api/download")
}

func (s *Server) handleMkdirRelay(w http.ResponseWriter, r *http.Request) {
	s.relayGET(w, r, "/api/mkdir", "/api/mkdir")
}

// silence unused in some builds
var _ = io.EOF
var _ = fmt.Sprintf
