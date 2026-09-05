//go:build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8765", "server")
	pin := flag.String("pin", "123456", "pin")
	flag.Parse()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	_ = c.WriteJSON(map[string]string{"type": "auth", "pin": *pin})
	_, raw, _ := c.ReadMessage()
	var auth struct{ OK bool `json:"ok"` }
	_ = json.Unmarshal(raw, &auth)
	if !auth.OK {
		log.Fatal("auth fail")
	}
	// send Chinese text
	_ = c.WriteJSON(map[string]interface{}{"type": "text", "text": "你好Hello"})
	_ = c.WriteJSON(map[string]interface{}{"type": "key", "key": "Enter", "down": true})
	_ = c.WriteJSON(map[string]interface{}{"type": "key", "key": "Enter", "down": false})
	time.Sleep(200 * time.Millisecond)
	fmt.Println("TEXT_SENT")
}
