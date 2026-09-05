// lan-remote-client: controller only — list devices on hub, remote-control them.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"lan-remote/internal/appwin"
	"lan-remote/internal/config"
	"lan-remote/internal/registry"
)

//go:embed ui.html
var uiHTML string

const appVersion = "1.1.0"

func main() {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Data{HTTPPort: 8765, RegistryPort: 8760}
	}

	hub := flag.String("hub", cfg.Hub, "registry hub host:port (required)")
	uiPort := flag.Int("ui", 0, "local UI port (0 = random)")
	noGUI := flag.Bool("no-gui", false, "open UI in browser")
	flag.Parse()

	hubAddr := *hub
	if hubAddr == "" {
		// try common: this machine's registry
		hubAddr = fmt.Sprintf("%s:%d", "127.0.0.1", cfg.RegistryPort)
		log.Println("no -hub given, using", hubAddr)
	}
	cfg.Hub = hubAddr
	_ = config.Save(cfg)

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
			"hub":     hubAddr,
			"version": appVersion,
		})
	})
	mux.HandleFunc("/api/devices", func(w http.ResponseWriter, r *http.Request) {
		devs, err := registry.FetchDevices(hubAddr)
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
	fmt.Printf("  Hub: %s\n", hubAddr)
	fmt.Printf("  UI:  %s\n", uiURL)
	fmt.Println("========================================")

	// probe hub
	go func() {
		time.Sleep(400 * time.Millisecond)
		if _, err := registry.FetchDevices(hubAddr); err != nil {
			log.Println("hub not reachable yet:", err)
		}
	}()

	if *noGUI {
		appwin.OpenBrowser(uiURL)
		appwin.WaitSignal()
		return
	}
	if !appwin.OpenWindow("LAN Remote", uiURL, 1100, 760) {
		appwin.OpenBrowser(uiURL)
		appwin.WaitSignal()
	}
}

// keep embed used even if we later inline
var _ embed.FS
var _ = os.Stderr
