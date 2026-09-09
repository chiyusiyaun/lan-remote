package portal

import (
	_ "embed"
	"encoding/json"
	"log"
	"net"
	"net/http"

	"lan-remote/internal/registry"
)

//go:embed index.html
var indexHTML string

// Config for the unified portal (usually on the server host :8765).
type Config struct {
	Addr     string
	Registry string // registry base, e.g. http://127.0.0.1:8765 or host:port
	Version  string
}

type Server struct {
	cfg Config
}

func New(cfg Config) *Server {
	if cfg.Version == "" {
		cfg.Version = "1.2.3"
	}
	return &Server{cfg: cfg}
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

// RegisterRoutes: device list + UI only (no WS/file proxy).
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/peers", s.handlePeers)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) ServeIndex() http.HandlerFunc { return s.handleIndex }

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"app":     "portal",
		"role":    "portal",
		"hub":     s.cfg.Registry,
		"proxy":   false,
		"version": s.cfg.Version,
	})
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	devs, err := registry.FetchDevices(s.cfg.Registry)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"peers": devs})
}
