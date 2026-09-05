// lan-remote-client: controller only — enter service IP in UI, list devices, remotes.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"lan-remote/internal/appwin"
	"lan-remote/internal/config"
	"lan-remote/internal/registry"
)

//go:embed ui.html
var uiHTML string

const appVersion = "1.2.0"

type hubState struct {
	mu  sync.RWMutex
	hub string
}

func (h *hubState) get() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hub
}

func (h *hubState) set(v string) {
	h.mu.Lock()
	h.hub = v
	h.mu.Unlock()
}

func normalizeHub(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimSuffix(s, "/")
	// host only -> add default registry port
	if !strings.Contains(s, ":") {
		s = s + ":8760"
	}
	return s
}

func main() {
	cfg, err := config.Load("client")
	if err != nil {
		cfg = &config.Data{RegistryPort: 8760}
	}

	hubFlag := flag.String("hub", "", "optional hub host[:port]; can also set in UI")
	uiPort := flag.Int("ui", 0, "local UI port (0 = random)")
	noGUI := flag.Bool("no-gui", false, "open UI in browser")
	flag.Parse()

	hs := &hubState{}
	// flag > saved config > empty (UI will ask)
	switch {
	case *hubFlag != "":
		hs.set(normalizeHub(*hubFlag))
	case cfg.Hub != "":
		hs.set(normalizeHub(cfg.Hub))
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *uiPort))
	if err != nil {
		appwin.Pause("UI listen failed: " + err.Error())
		return
	}
	uiURL := fmt.Sprintf("http://%s/", ln.Addr().String())

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(uiHTML))
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"app":     "client",
			"hub":     hs.get(),
			"version": appVersion,
		})
	})
	// set / change service (hub) address from UI
	mux.HandleFunc("/api/hub", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}
		var body struct {
			Hub string `json:"hub"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		addr := normalizeHub(body.Hub)
		if addr == "" || strings.HasPrefix(addr, ":") {
			http.Error(w, "invalid service address", 400)
			return
		}
		// probe registry
		if _, err := registry.FetchDevices(addr); err != nil {
			http.Error(w, "无法连接注册中心 "+addr+"，请检查 IP 与端口", 502)
			return
		}
		hs.set(addr)
		cfg.Hub = addr
		_ = config.Save("client", cfg)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "hub": addr})
	})
	mux.HandleFunc("/api/devices", func(w http.ResponseWriter, r *http.Request) {
		hub := hs.get()
		if hub == "" {
			http.Error(w, "hub not set", 400)
			return
		}
		devs, err := registry.FetchDevices(hub)
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"devices": devs})
	})

	go func() {
		if err := http.Serve(ln, mux); err != nil {
			log.Println("ui:", err)
		}
	}()

	fmt.Println("========================================")
	fmt.Println("  LAN Remote CLIENT  v" + appVersion)
	if h := hs.get(); h != "" {
		fmt.Printf("  Hub: %s (saved)\n", h)
	} else {
		fmt.Println("  Hub: (enter Service IP in the window)")
	}
	fmt.Printf("  UI:  %s\n", uiURL)
	fmt.Println("========================================")

	if *noGUI {
		appwin.OpenBrowser(uiURL)
		appwin.WaitSignal()
		return
	}
	if !appwin.OpenWindow("LAN Remote Client", uiURL, 1100, 760) {
		appwin.OpenBrowser(uiURL)
		appwin.WaitSignal()
	}
}
