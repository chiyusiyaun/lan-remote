// lan-remote-client: every PC — can be controlled (screen+input) and can control others.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"lan-remote/internal/appwin"
	"lan-remote/internal/config"
	"lan-remote/internal/discovery"
	"lan-remote/internal/registry"
	"lan-remote/internal/server"
)

const appVersion = "1.2.0"

func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		if u, err := user.Current(); err == nil {
			return u.Username
		}
		return "LAN-Remote"
	}
	return h
}

// normalizeHub accepts host, host:port, http://host, http://host:port/path, https://...
func normalizeHub(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	hasScheme := strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
	if !hasScheme {
		s = "http://" + s
	}
	// bare host without port and without path → default registry port
	rest := s
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	slash := strings.Index(rest, "/")
	hostPart := rest
	path := ""
	if slash >= 0 {
		hostPart = rest[:slash]
		path = rest[slash:]
	}
	if hostPart != "" && !strings.Contains(hostPart, ":") {
		// no port — only add :8760 when there is no path prefix (path-based deploys often hide the port)
		if path == "" || path == "/" {
			s = "http://" + hostPart + ":8760"
		} else {
			s = "http://" + hostPart + path
		}
	}
	return strings.TrimRight(s, "/")
}

type hubBox struct {
	mu  sync.RWMutex
	hub string
}

func (h *hubBox) get() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hub
}

func (h *hubBox) set(v string) {
	h.mu.Lock()
	h.hub = v
	h.mu.Unlock()
}

func main() {
	cfg, err := config.Load("client")
	if err != nil {
		cfg = &config.Data{HTTPPort: 8765, RegistryPort: 8760, Quality: 70, FPS: 15}
	}

	httpPort := flag.Int("port", cfg.HTTPPort, "control port (this PC is controllable here)")
	pin := flag.String("pin", cfg.PIN, "PIN for this PC")
	hub := flag.String("hub", cfg.Hub, "Service/registry IP (can also set in UI)")
	quality := flag.Int("q", cfg.Quality, "JPEG quality 1-100")
	fps := flag.Int("fps", cfg.FPS, "FPS 1-240")
	noGUI := flag.Bool("no-gui", false, "console + browser")
	bg := flag.Bool("bg", false, "background: log to file, no console")
	logPath := flag.String("log", "", "log file path when -bg")
	flag.Parse()

	if *bg {
		appwin.Background(*logPath)
	}

	name := hostname()
	if cfg.DeviceName != "" {
		name = cfg.DeviceName
	}
	ip := discovery.PrimaryIP()

	if *pin != "" {
		cfg.PIN = *pin
	}
	cfg.HTTPPort = *httpPort
	cfg.Quality = *quality
	cfg.FPS = *fps
	if *hub != "" {
		cfg.Hub = normalizeHub(*hub)
	}
	if cfg.DeviceName == "" {
		cfg.DeviceName = name
	}
	_ = config.Save("client", cfg)

	hb := &hubBox{hub: normalizeHub(cfg.Hub)}
	var regClient *registry.Client

	startReg := func(hubAddr string) {
		if hubAddr == "" {
			return
		}
		if regClient != nil {
			regClient.Stop()
		}
		regClient = registry.NewClientMulti(hubAddr, name, discovery.AllIPs(), *httpPort, cfg.PIN != "", appVersion)
		regClient.Start()
	}

	// if hub already saved, register immediately (port may not be final yet; re-register after bind)
	if h := hb.get(); h != "" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			startReg(h)
		}()
	}

	srv := server.New(server.Config{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		PIN:     cfg.PIN,
		Quality: cfg.Quality,
		FPS:     cfg.FPS,
		Extra: map[string]string{
			"hub":     hb.get(),
			"role":    "client",
			"version": appVersion,
			"app":     "client",
		},
		Peers: func() interface{} {
			h := hb.get()
			if h == "" {
				return []interface{}{}
			}
			devs, err := registry.FetchDevices(h)
			if err != nil {
				return []interface{}{}
			}
			return devs
		},
		OnPINChanged: func(p string) {
			cfg.PIN = p
			_ = config.Save("client", cfg)
			if regClient != nil {
				regClient.SetPINSet(p != "")
			}
		},
		OnSetHub: func(raw string) error {
			addr := normalizeHub(raw)
			if addr == "" {
				hb.set("")
				cfg.Hub = ""
				_ = config.Save("client", cfg)
				return nil
			}
			if _, err := registry.FetchDevices(addr); err != nil {
				return fmt.Errorf("无法连接 Service %s", addr)
			}
			hb.set(addr)
			cfg.Hub = addr
			_ = config.Save("client", cfg)
			// re-bind registry with actual port
			startReg(addr)
			log.Println("hub set:", addr)
			return nil
		},
	})

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	var port int
	for i := 0; i < 60; i++ {
		if srv.Port() > 0 {
			port = srv.Port()
			break
		}
		select {
		case err := <-errCh:
			appwin.Pause("Client failed: " + err.Error())
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
	if port == 0 {
		port = *httpPort
	}
	// correct registry port if bind changed
	if regClient != nil {
		startReg(hb.get())
	}

	localURL := fmt.Sprintf("http://127.0.0.1:%d/", port)

	fmt.Println("========================================")
	fmt.Println("  LAN Remote CLIENT  v" + appVersion)
	fmt.Printf("  Device:  %s\n", name)
	fmt.Printf("  Control: http://%s:%d  (this PC)\n", ip, port)
	if h := hb.get(); h != "" {
		fmt.Printf("  Service: %s\n", h)
	} else {
		fmt.Println("  Service: (enter in UI)")
	}
	if cfg.PIN != "" {
		fmt.Printf("  PIN:     %s (saved)\n", cfg.PIN)
	} else {
		fmt.Println("  PIN:     (set in UI)")
	}
	fmt.Printf("  Config:  %s\n", config.Path("client"))
	fmt.Println("========================================")

	go func() {
		if err, ok := <-errCh; ok && err != nil {
			if !*bg {
				appwin.Pause("Client crashed: " + err.Error())
			}
			log.Println("client error:", err)
			os.Exit(1)
		}
	}()

	if *noGUI {
		appwin.OpenBrowser(localURL)
		appwin.WaitSignal()
		return
	}
	if *bg {
		go func() {
			time.Sleep(300 * time.Millisecond)
			appwin.RunWithTray("LAN Remote Client", localURL, 1100, 760, true, appwin.IconClient)
		}()
		appwin.WaitSignal()
		return
	}
	appwin.RunWithTray("LAN Remote Client", localURL, 1100, 760, false, appwin.IconClient)
}
