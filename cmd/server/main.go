// lan-remote-server: control endpoint (screen+input) + optional registry hub.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"time"

	"lan-remote/internal/appwin"
	"lan-remote/internal/config"
	"lan-remote/internal/discovery"
	"lan-remote/internal/registry"
	"lan-remote/internal/server"
)

const appVersion = "1.1.0"

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

func main() {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Data{HTTPPort: 8765, RegistryPort: 8760, Quality: 70, FPS: 15}
	}

	httpPort := flag.Int("port", cfg.HTTPPort, "control port (stream + input)")
	regPort := flag.Int("registry", cfg.RegistryPort, "registry port (when this node is hub)")
	hub := flag.String("hub", cfg.Hub, "registry hub host:port; empty = this node is the hub")
	pin := flag.String("pin", cfg.PIN, "PIN (saved to config)")
	quality := flag.Int("q", cfg.Quality, "JPEG quality 1-100")
	fps := flag.Int("fps", cfg.FPS, "stream FPS 1-60")
	noGUI := flag.Bool("no-gui", false, "console only")
	hubOnly := flag.Bool("hub-only", false, "registry hub only (no screen/input)")
	flag.Parse()

	name := hostname()
	if cfg.DeviceName != "" {
		name = cfg.DeviceName
	}
	ip := discovery.PrimaryIP()

	if *pin != "" {
		cfg.PIN = *pin
	}
	cfg.Hub = *hub
	cfg.HTTPPort = *httpPort
	cfg.RegistryPort = *regPort
	cfg.Quality = *quality
	cfg.FPS = *fps
	if cfg.DeviceName == "" {
		cfg.DeviceName = name
	}
	_ = config.Save(cfg)

	hubAddr := cfg.Hub
	isHub := hubAddr == ""
	if isHub {
		hubAddr = fmt.Sprintf("%s:%d", ip, *regPort)
		reg := registry.New(*regPort)
		go func() {
			if err := reg.ListenAndServe(); err != nil {
				log.Println("registry:", err)
			}
		}()
		time.Sleep(120 * time.Millisecond)
	}

	if *hubOnly {
		fmt.Printf("Hub only. Registry: http://%s:%d\n", ip, *regPort)
		appwin.WaitSignal()
		return
	}

	var regClient *registry.Client
	srv := server.New(server.Config{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		PIN:     cfg.PIN,
		Quality: cfg.Quality,
		FPS:     cfg.FPS,
		Extra: map[string]string{
			"hub":      hubAddr,
			"role":     roleOf(isHub),
			"version":  appVersion,
			"app":      "server",
		},
		Peers: func() interface{} {
			devs, err := registry.FetchDevices(hubAddr)
			if err != nil {
				return []interface{}{}
			}
			return devs
		},
		OnPINChanged: func(p string) {
			cfg.PIN = p
			_ = config.Save(cfg)
			if regClient != nil {
				regClient.SetPINSet(p != "")
			}
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
			appwin.Pause("Server failed: " + err.Error())
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
	if port == 0 {
		port = *httpPort
	}

	regClient = registry.NewClient(hubAddr, name, ip, port, cfg.PIN != "", appVersion)
	regClient.Start()

	localURL := fmt.Sprintf("http://127.0.0.1:%d/", port)

	fmt.Println("========================================")
	fmt.Println("  LAN Remote SERVER  v" + appVersion)
	fmt.Printf("  Device:   %s\n", name)
	fmt.Printf("  Control:  http://%s:%d\n", ip, port)
	if isHub {
		fmt.Printf("  Registry: http://%s:%d  (HUB)\n", ip, *regPort)
	} else {
		fmt.Printf("  Hub:      %s\n", hubAddr)
	}
	fmt.Printf("  PIN:      %s\n", pinDisplay(cfg.PIN))
	fmt.Printf("  Config:   %s\n", config.Path())
	fmt.Println("========================================")

	go func() {
		if err, ok := <-errCh; ok && err != nil {
			appwin.Pause("Server crashed: " + err.Error())
			os.Exit(1)
		}
	}()

	if *noGUI {
		appwin.OpenBrowser(localURL)
		appwin.WaitSignal()
		return
	}
	if !appwin.OpenWindow("LAN Remote Server", localURL, 1000, 720) {
		appwin.OpenBrowser(localURL)
		appwin.WaitSignal()
	}
}

func roleOf(isHub bool) string {
	if isHub {
		return "hub"
	}
	return "node"
}

func pinDisplay(p string) string {
	if p == "" {
		return "(not set)"
	}
	return p + "  (saved)"
}
