package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"runtime"
	"syscall"
	"time"

	"lan-remote/internal/config"
	"lan-remote/internal/discovery"
	"lan-remote/internal/registry"
	"lan-remote/internal/server"
)

const appVersion = "1.1.0"

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

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

	httpPort := flag.Int("port", cfg.HTTPPort, "control (stream/input) port")
	regPort := flag.Int("registry", cfg.RegistryPort, "registry port (when this node is hub)")
	hub := flag.String("hub", cfg.Hub, "registry hub host:port; empty = this node is the hub")
	pin := flag.String("pin", cfg.PIN, "PIN (saved to config if set via GUI/flag)")
	quality := flag.Int("q", cfg.Quality, "JPEG quality 1-100")
	fps := flag.Int("fps", cfg.FPS, "stream FPS 1-60")
	noGUI := flag.Bool("no-gui", false, "console only, no desktop window")
	hubOnly := flag.Bool("hub-only", false, "run registry hub only (no control/GUI)")
	flag.Parse()

	name := hostname()
	if cfg.DeviceName != "" {
		name = cfg.DeviceName
	}
	ip := discovery.PrimaryIP()

	// persist flag overrides
	if *pin != cfg.PIN && *pin != "" {
		cfg.PIN = *pin
	}
	if *hub != cfg.Hub {
		cfg.Hub = *hub
	}
	cfg.HTTPPort = *httpPort
	cfg.RegistryPort = *regPort
	cfg.Quality = *quality
	cfg.FPS = *fps
	if cfg.DeviceName == "" {
		cfg.DeviceName = name
	}
	_ = config.Save(cfg)

	// resolve hub address
	hubAddr := cfg.Hub
	isHub := hubAddr == ""
	if isHub {
		hubAddr = fmt.Sprintf("%s:%d", ip, *regPort)
	}

	// --- Port 1: registry ---
	if isHub {
		reg := registry.New(*regPort)
		go func() {
			if err := reg.ListenAndServe(); err != nil {
				log.Println("registry:", err)
			}
		}()
		// wait briefly
		time.Sleep(150 * time.Millisecond)
		log.Printf("hub registry :%d", *regPort)
	}

	if *hubOnly {
		fmt.Printf("Hub only. Registry: http://%s:%d  Control nodes register here.\n", ip, *regPort)
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		return
	}

	// --- Port 2: control (screen + input + UI) ---
	var regClient *registry.Client
	srv := server.New(server.Config{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		PIN:     cfg.PIN,
		Quality: cfg.Quality,
		FPS:     cfg.FPS,
		Extra: map[string]string{
			"hub":      hubAddr,
			"registry": fmt.Sprintf("%d", *regPort),
			"role":     roleOf(isHub),
			"version":  appVersion,
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
			if err := config.Save(cfg); err != nil {
				log.Println("save pin:", err)
			} else {
				log.Println("PIN saved to", config.Path())
			}
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
			pause("启动失败: " + err.Error() + "\n(端口可能被占用，请关闭已运行的 LAN Remote)")
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
	if port == 0 {
		port = *httpPort
	}

	// register this node to hub
	regClient = registry.NewClient(hubAddr, name, ip, port, cfg.PIN != "", appVersion)
	regClient.Start()

	localURL := fmt.Sprintf("http://127.0.0.1:%d/", port)

	fmt.Println("========================================")
	fmt.Println("  LAN Remote Desktop  v" + appVersion)
	fmt.Printf("  Device:  %s\n", name)
	fmt.Printf("  Control: http://%s:%d\n", ip, port)
	if isHub {
		fmt.Printf("  Registry: http://%s:%d  (this node is HUB)\n", ip, *regPort)
	} else {
		fmt.Printf("  Hub:     %s\n", hubAddr)
	}
	if cfg.PIN != "" {
		fmt.Printf("  PIN:     %s  (saved)\n", cfg.PIN)
	} else {
		fmt.Println("  PIN:     (not set — will ask in UI, then save)")
	}
	fmt.Printf("  Config:  %s\n", config.Path())
	fmt.Println("========================================")

	go func() {
		if err, ok := <-errCh; ok && err != nil {
			pause("服务异常退出: " + err.Error())
			os.Exit(1)
		}
	}()

	if *noGUI {
		openBrowser(localURL)
		waitSignal()
		return
	}

	ok := openAppWindow("LAN Remote", localURL, 1100, 760)
	if !ok {
		if runtime.GOOS != "windows" {
			log.Println("Native window unavailable, opening browser")
			openBrowser(localURL)
			waitSignal()
			return
		}
		pause("无法创建原生窗口（需要 WebView2 运行时）。\n可安装 WebView2，或使用 -no-gui")
		return
	}
}

func roleOf(isHub bool) string {
	if isHub {
		return "hub"
	}
	return "node"
}

func waitSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
