// lan-remote-server: registry (8760) + unified portal (8765). Not a controllable device.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	portalPort := flag.Int("portal", cfg.HTTPPort, "portal/UI listen port (no self-register)")
	flag.Parse()

	cfg.RegistryPort = *regPort
	cfg.HTTPPort = *portalPort
	cfg.Hub = ""
	cfg.PIN = "" // server is never controllable
	_ = config.Save("service", cfg)

	ip := discovery.PrimaryIP()
	reg := registry.New(*regPort)

	errCh := make(chan error, 2)
	go func() {
		if err := reg.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("registry: %w", err)
		}
	}()

	// Portal: device list + remote control via WS proxy. Does NOT register itself.
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

	fmt.Println("========================================")
	fmt.Println("  LAN Remote SERVER  v" + appVersion)
	fmt.Println("  Role:     Registry + Portal (not controllable)")
	fmt.Printf("  Registry: :%d  ->  http://%s:%d\n", *regPort, ip, *regPort)
	fmt.Printf("  Portal:   :%d  ->  http://%s:%d  (统一入口)\n", *portalPort, ip, *portalPort)
	fmt.Println("  Clients register on 8760; users open Portal on 8765")
	fmt.Println("========================================")

	go func() {
		if err := <-errCh; err != nil {
			log.Println(err)
			fmt.Println("Port may be in use. Press Enter to exit.")
			var b [1]byte
			_, _ = os.Stdin.Read(b[:])
			os.Exit(1)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	fmt.Println("bye")
}
