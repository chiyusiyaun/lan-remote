// lan-remote-server: single port 鈥?registry + portal. Not a controllable device.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"lan-remote/internal/appwin"
	"lan-remote/internal/config"
	"lan-remote/internal/discovery"
	"lan-remote/internal/portal"
	"lan-remote/internal/registry"
)

const appVersion = "1.2.3"

func main() {
	cfg, err := config.Load("service")
	if err != nil || cfg == nil {
		cfg = &config.Data{HTTPPort: 8765, RegistryPort: 8765}
	}
	port := flag.Int("port", 8765, "unified listen port (registry + portal)")
	noGUI := flag.Bool("no-gui", false, "no window, console only")
	bg := flag.Bool("bg", false, "background: log to file, no console")
	logPath := flag.String("log", "", "log file path when -bg")
	flag.Parse()

	if *bg {
		appwin.Background(*logPath)
	}

	cfg.HTTPPort = *port
	cfg.RegistryPort = *port
	cfg.Hub = ""
	cfg.PIN = ""
	_ = config.Save("service", cfg)

	ip := discovery.PrimaryIP()

	reg := registry.New(*port)
	p := portal.New(portal.Config{
		Addr:     fmt.Sprintf(":%d", *port),
		Registry: fmt.Sprintf("127.0.0.1:%d", *port),
		Version:  appVersion,
	})

	inner := http.NewServeMux()
	inner.HandleFunc("/", p.ServeIndex())
	reg.RegisterRoutes(inner)
	p.RegisterRoutes(inner)

	outer := http.NewServeMux()
	outer.Handle("/server/", http.StripPrefix("/server", inner))
	outer.HandleFunc("/server", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/server/", http.StatusFound)
	})
	outer.Handle("/", inner)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		msg := fmt.Sprintf("listen :%d failed: %v (port may be in use)", *port, err)
		if !*bg {
			appwin.Pause(msg)
		} else {
			log.Println(msg)
		}
		os.Exit(1)
	}

	srv := &http.Server{Handler: outer}
	errCh := make(chan error, 1)
	go func() {
		reg.StartSweep()
		if err := srv.Serve(ln); err != nil {
			errCh <- err
		}
	}()

	url := fmt.Sprintf("http://%s:%d/", ip, *port)
	admin := fmt.Sprintf("http://127.0.0.1:%d/server/", *port)

	fmt.Println("========================================")
	fmt.Println("  LAN Remote SERVER  v" + appVersion)
	fmt.Println("  Role:     Registry + Portal (single port)")
	fmt.Printf("  Port:     %d\n", *port)
	fmt.Printf("  Portal:   %s\n", url)
	fmt.Printf("  Admin:    %s\n", admin)
	fmt.Printf("  Service:  http://%s:%d/server\n", ip, *port)
	fmt.Println("========================================")

	go func() {
		if err := <-errCh; err != nil {
			log.Println(err)
			if !*bg {
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
		go func() {
			time.Sleep(300 * time.Millisecond)
			appwin.RunWithTray("LAN Remote Server", admin, 960, 680, true, appwin.IconServer)
		}()
		appwin.WaitSignal()
		return
	}
	appwin.RunWithTray("LAN Remote Server", admin, 960, 680, false, appwin.IconServer)
}
