// Registry admin UI — simple status page on the service port.
package registry

import (
	_ "embed"
	"encoding/json"
	"net/http"
)

//go:embed admin.html
var adminHTML string

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminHTML))
}

func (s *Server) handleAdminAPI(w http.ResponseWriter, r *http.Request) {
	s.handleDevices(w, r)
}

var _ = json.Marshal
