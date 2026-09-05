# LAN Remote Desktop — build scripts
.PHONY: all win linux clean test

OUT=dist
LDFLAGS=-s -w

all: win linux

win:
	go build -ldflags "$(LDFLAGS)" -o $(OUT)/lan-remote-windows-amd64.exe .

linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUT)/lan-remote-linux-amd64 .

arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUT)/lan-remote-linux-arm64 .

test:
	go test ./...

clean:
	rm -rf $(OUT)
