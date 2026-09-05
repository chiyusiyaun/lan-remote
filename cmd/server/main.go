// lan-remote-server: registry + portal. Not a controllable device.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"lan-remote/internal/appwin"
	"lan-remote/internal/config"
	"lan-remote/internal/discovery"
	"lan-remote/internal/portal"
	"lan-remote/internal/registry"
)

const appVersion = "1.2.0"

func main() {
	cfg, err := config.Load("service")
	if err != nil || cfg == nil {
		cfg = &config.Data{RegistryPort: 8760, HTTPPort: 8765}
	}
	regPort := flag.Int("port", cfg.RegistryPort, "registry listen port")
	portalPort := flag.Int("portal", cfg.HTTPPort, "portal/UI listen port")
	noGUI := flag.Bool("no-gui", false, "no window, console only")
	bg := flag.Bool("bg", false, "background: log to file, no console (daemon-friendly)")
	logPath := flag.String("log", "", "log file path when -bg")
	flag.Parse()

	if *bg {
		appwin.Background(*logPath)
	}

	cfg.RegistryPort = *regPort
	cfg.HTTPPort = *portalPort
	cfg.Hub = ""
	cfg.PIN = ""
	_ = config.Save("service", cfg)

	ip := discovery.PrimaryIP()
	reg := registry.New(*regPort)

	errCh := make(chan error, 2)
	go func() {
		if err := reg.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("registry: %w", err)
		}
	}()

	p := portal.New(portal.Config{
		Addr:     fmt.Sprintf(":%d", *portalPort),
		Registry: fmt.Sprintf("127.0.0.1:%d", *regPort),
		Version:  appVersion,
	})
	go func() {
		if err := p.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("portal: %w", err)
		}
	}()

	adminURL := fmt.Sprintf("http://127.0.0.1:%d/", *regPort)
	portalURL := fmt.Sprintf("http://%s:%d/", ip, *portalPort)

	fmt.Println("========================================")
	fmt.Println("  LAN Remote SERVER  v" + appVersion)
	fmt.Println("  Role:     Registry + Portal")
	fmt.Printf("  Admin:    %s\n", adminURL)
	fmt.Printf("  Portal:   %s\n", portalURL)
	fmt.Println("========================================")

	go func() {
		if err := <-errCh; err != nil {
			log.Println(err)
			if !*bg && !*noGUI {
				appwin.Pause("Server failed: " + err.Error())
			}
			os.Exit(1)
		}
	}()

	if *noGUI {
		appwin.WaitSignal()
		return
	}

	if *bg {
		// headless: tray only (Windows/Linux desktop); else just wait
		go func() {
			time.Sleep(300 * time.Millisecond)
			appwin.RunWithTray("LAN Remote Server", adminURL, 900, 640, true)
		}()
		appwin.WaitSignal()
		return
	}

	// GUI: window + tray (close window → minimize to tray)
	appwin.RunWithTray("LAN Remote Server", adminURL, 960, 680, false)
}
