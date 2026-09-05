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
		cfg.Version = "1.2.0"
	}
	return &Server{
		cfg: cfg,
		upg: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/peers", s.handlePeers)
	mux.HandleFunc("/api/file", s.handleFileRelay)
	mux.HandleFunc("/api/files", s.handleFilesRelay)
	mux.HandleFunc("/api/download", s.handleDownloadRelay)
	mux.HandleFunc("/proxy", s.handleProxy)

	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	log.Printf("portal listening on %s", ln.Addr().String())
	return http.Serve(ln, mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

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
// Browser:  ws://server:8765/proxy?target=192.168.1.20:8765
// Upstream: ws://192.168.1.20:8765/ws
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "target required", 400)
		return
	}
	target = strings.TrimSpace(target)
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
		_ = clientConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"auth","ok":false,"error":"upstream"}`))
		return
	}
	defer upConn.Close()

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

// handleFileRelay proxies POST /api/file?target=host:port to the client.
func (s *Server) handleFileRelay(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "target required", 400)
		return
	}
	if !strings.Contains(target, ":") {
		target += ":8765"
	}
	pin := r.Header.Get("X-LR-Pin")
	upURL := "http://" + target + "/api/file"
	if pin != "" {
		// keep pin header
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

func (s *Server) relayGET(w http.ResponseWriter, r *http.Request, upstreamPath string) {
	target := r.URL.Query().Get("target")
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
	s.relayGET(w, r, "/api/files")
}

func (s *Server) handleDownloadRelay(w http.ResponseWriter, r *http.Request) {
	s.relayGET(w, r, "/api/download")
}

// silence unused in some builds
var _ = io.EOF
var _ = fmt.Sprintf
