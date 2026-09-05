// lan-remote-server: registry hub only. Devices register here; clients fetch the list.
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
	"lan-remote/internal/registry"
)

const appVersion = "1.2.0"

func main() {
	cfg, err := config.Load("service")
	if err != nil || cfg == nil {
		cfg = &config.Data{RegistryPort: 8760}
	}
	port := flag.Int("port", cfg.RegistryPort, "registry listen port")
	flag.Parse()

	cfg.RegistryPort = *port
	cfg.Hub = "" // this is the hub
	_ = config.Save("service", cfg)

	ip := discovery.PrimaryIP()
	reg := registry.New(*port)

	errCh := make(chan error, 1)
	go func() {
		if err := reg.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	fmt.Println("========================================")
	fmt.Println("  LAN Remote SERVER  v" + appVersion)
	fmt.Println("  Role:     Registry / Service")
	fmt.Printf("  Listen:   :%d\n", *port)
	fmt.Printf("  Address:  http://%s:%d\n", ip, *port)
	fmt.Println("  Clients register here on port", *port)
	fmt.Println("========================================")

	go func() {
		if err := <-errCh; err != nil {
			log.Println("registry failed:", err)
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
