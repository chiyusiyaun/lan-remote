//go:build ignore

package main

// Builds a Debian package from dist/ Linux binaries + packaging/linux assets.
// Usage: go run tools/mkdeb/main.go -out dist/lan-remote_1.2.0_amd64.deb
import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	version = "1.2.0"
	arch    = "amd64"
)

type fileEntry struct {
	src  string
	dst  string // path inside package, e.g. /usr/bin/lan-remote-client
	mode int64
}

func main() {
	out := flag.String("out", "dist/lan-remote_"+version+"_"+arch+".deb", "output deb path")
	flag.Parse()

	root := "."
	entries := []fileEntry{
		{root + "/dist/lan-remote-client-linux", "/usr/bin/lan-remote-client", 0755},
		{root + "/dist/lan-remote-server-linux", "/usr/bin/lan-remote-server", 0755},
		{root + "/packaging/linux/lan-remote-client.desktop", "/usr/share/applications/lan-remote-client.desktop", 0644},
		{root + "/packaging/linux/lan-remote-server.desktop", "/usr/share/applications/lan-remote-server.desktop", 0644},
		{root + "/packaging/linux/icon-client.png", "/usr/share/icons/hicolor/256x256/apps/lan-remote-client.png", 0644},
		{root + "/packaging/linux/icon-server.png", "/usr/share/icons/hicolor/256x256/apps/lan-remote-server.png", 0644},
	}

	var missing []string
	for _, e := range entries {
		if _, err := os.Stat(e.src); err != nil {
			missing = append(missing, e.src)
		}
	}
	if len(missing) > 0 {
		fmt.Println("missing files:")
		for _, m := range missing {
			fmt.Println(" ", m)
		}
		os.Exit(1)
	}

	// control
	control := strings.Join([]string{
		"Package: lan-remote",
		"Version: " + version,
		"Section: net",
		"Priority: optional",
		"Architecture: " + arch,
		"Maintainer: LAN Remote <dev@localhost>",
		"Depends: libc6",
		"Recommends: xdotool, scrot | imagemagick",
		"Description: LAN remote desktop (registry + client)",
		" Registry hub (8760) and portal (8765) plus per-PC client",
		" for LAN remote control, file transfer, and device discovery.",
		"",
	}, "\n")

	controlTar := buildTar([]tarFile{
		{name: "./control", body: []byte(control), mode: 0644},
	})
	dataTar := buildTarFromEntries(entries)

	if err := os.MkdirAll(filepath.Dir(*out), 0755); err != nil {
		fatal(err)
	}
	if err := writeDeb(*out, controlTar, dataTar); err != nil {
		fatal(err)
	}
	st, _ := os.Stat(*out)
	fmt.Println("wrote", *out, st.Size(), "bytes")
}

type tarFile struct {
	name string
	body []byte
	mode int64
}

func buildTar(files []tarFile) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	now := time.Now()
	for _, f := range files {
		hdr := &tar.Header{
			Name:    f.name,
			Mode:    f.mode,
			Size:    int64(len(f.body)),
			ModTime: now,
			Format:  tar.FormatGNU,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			fatal(err)
		}
		if _, err := tw.Write(f.body); err != nil {
			fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		fatal(err)
	}
	if err := gz.Close(); err != nil {
		fatal(err)
	}
	return buf.Bytes()
}

func buildTarFromEntries(entries []fileEntry) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	now := time.Now()
	// ensure parent dirs exist in archive
	dirs := map[string]bool{}
	addDir := func(p string) {
		for {
			p = filepath.ToSlash(filepath.Dir(p))
			if p == "/" || p == "." || p == "" {
				return
			}
			if dirs[p] {
				return
			}
			dirs[p] = true
			hdr := &tar.Header{
				Name:     p + "/",
				Typeflag: tar.TypeDir,
				Mode:     0755,
				ModTime:  now,
				Format:   tar.FormatGNU,
			}
			_ = tw.WriteHeader(hdr)
		}
	}
	for _, e := range entries {
		data, err := os.ReadFile(e.src)
		if err != nil {
			fatal(err)
		}
		name := e.dst
		if !strings.HasPrefix(name, "/") {
			name = "/" + name
		}
		addDir(name)
		hdr := &tar.Header{
			Name:    strings.TrimPrefix(name, "/"),
			Mode:    e.mode,
			Size:    int64(len(data)),
			ModTime: now,
			Format:  tar.FormatGNU,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		fatal(err)
	}
	if err := gz.Close(); err != nil {
		fatal(err)
	}
	return buf.Bytes()
}

func writeDeb(path string, controlTar, dataTar []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// ar archive with debian-binary, control.tar.gz, data.tar.gz
	debianBinary := []byte("2.0\n")
	if err := writeArFile(f, "debian-binary", debianBinary); err != nil {
		return err
	}
	if err := writeArFile(f, "control.tar.gz", controlTar); err != nil {
		return err
	}
	return writeArFile(f, "data.tar.gz", dataTar)
}

func writeArFile(w io.Writer, name string, data []byte) error {
	// global header only once — caller writes it
	return writeArMember(w, name, data)
}

var arMagicWritten bool

func writeArMember(w io.Writer, name string, data []byte) error {
	if !arMagicWritten {
		if _, err := w.Write([]byte("!<arch>\n")); err != nil {
			return err
		}
		arMagicWritten = true
	}
	// filename padded to 16
	n := name
	if len(n) > 15 {
		return fmt.Errorf("name too long: %s", name)
	}
	hdr := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8o%-10d`\n",
		n+"/", 0, 0, 0, 0100644, len(data))
	if _, err := io.WriteString(w, hdr); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if len(data)%2 == 1 {
		if _, err := w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
