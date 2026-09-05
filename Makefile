.PHONY: all server client linux clean

OUT=dist
LDFLAGS=-s -w
LDFLAGS_WIN=$(LDFLAGS) -H windowsgui

all: server client

server:
	go build -ldflags "$(LDFLAGS_WIN)" -o $(OUT)/lan-remote-server.exe ./cmd/server
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUT)/lan-remote-server-linux ./cmd/server

client:
	go build -ldflags "$(LDFLAGS_WIN)" -o $(OUT)/lan-remote-client.exe ./cmd/client
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUT)/lan-remote-client-linux ./cmd/client

test:
	go test ./...

clean:
	rm -rf $(OUT)
