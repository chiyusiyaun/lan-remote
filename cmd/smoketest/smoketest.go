//go:build ignore

package main

// Quick WS integration smoke test: start server externally, then:
//   go run cmd/smoketest/smoketest.go -addr 127.0.0.1:8765 -pin 123456

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8765", "server addr")
	pin := flag.String("pin", "123456", "pin")
	flag.Parse()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	_ = c.WriteJSON(map[string]string{"type": "auth", "pin": *pin})
	_, raw, err := c.ReadMessage()
	if err != nil {
		log.Fatal("read auth:", err)
	}
	var auth struct {
		Type string `json:"type"`
		OK   bool   `json:"ok"`
	}
	_ = json.Unmarshal(raw, &auth)
	if auth.Type != "auth" || !auth.OK {
		log.Fatalf("auth failed: %s", raw)
	}
	fmt.Println("AUTH_OK")

	// Expect at least one binary JPEG frame within 3s
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	n := 0
	for n < 3 {
		mt, data, err := c.ReadMessage()
		if err != nil {
			log.Fatal("frame:", err)
		}
		if mt != websocket.BinaryMessage {
			continue
		}
		if len(data) < 100 || data[0] != 0xFF || data[1] != 0xD8 {
			log.Fatalf("not a jpeg, head=%x", data[:4])
		}
		n++
		fmt.Printf("FRAME %d bytes=%d\n", n, len(data))
	}

	// Send a harmless mouse move + settings
	_ = c.WriteJSON(map[string]interface{}{"type": "move", "x": 10, "y": 10})
	_ = c.WriteJSON(map[string]interface{}{"type": "settings", "quality": 50, "fps": 10})
	fmt.Println("INPUT_SENT")
	fmt.Println("SMOKE_PASS")
	os.Exit(0)
}
