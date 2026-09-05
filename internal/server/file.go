package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// receiveDir returns a folder for incoming files (Downloads/lan-remote).
func receiveDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir(), nil
	}
	// Windows Downloads, Linux Downloads
	dl := filepath.Join(home, "Downloads")
	if _, err := os.Stat(dl); err != nil {
		dl = home
	}
	dir := filepath.Join(dl, "lan-remote")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func uniquePath(dir, name string) string {
	name = filepath.Base(name) // prevent path traversal
	if name == "" || name == "." || name == ".." {
		name = "file.bin"
	}
	dst := filepath.Join(dir, name)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return dst
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 1000; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s(%d)%s", base, i, ext))
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			return cand
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, time.Now().Unix(), ext))
}

// handleUpload receives a file from the controller (multipart or raw).
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	pin := r.Header.Get("X-LR-Pin")
	if pin == "" {
		pin = r.URL.Query().Get("pin")
	}
	if s.PIN() == "" || pin != s.PIN() {
		http.Error(w, "bad pin", 403)
		return
	}

	var reader io.Reader
	var filename string

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(256 << 20); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file field required", 400)
			return
		}
		defer f.Close()
		reader = f
		filename = hdr.Filename
	} else {
		reader = r.Body
		filename = r.URL.Query().Get("name")
		if filename == "" {
			filename = "upload.bin"
		}
	}

	dir, err := receiveDir()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	dst := uniquePath(dir, filename)
	out, err := os.Create(dst)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	n, err := io.Copy(out, reader)
	out.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	log.Printf("file received: %s (%d bytes)", dst, n)
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"ok":true,"path":%q,"size":%d}`, dst, n)
}

// handleDownload sends a file to the controller (by path, PIN required).
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	pin := r.Header.Get("X-LR-Pin")
	if pin == "" {
		pin = r.URL.Query().Get("pin")
	}
	if s.PIN() == "" || pin != s.PIN() {
		http.Error(w, "bad pin", 403)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", 400)
		return
	}
	// only allow files under Downloads/lan-remote
	dir, err := receiveDir()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	clean := filepath.Clean(path)
	if !strings.HasPrefix(clean, filepath.Clean(dir)+string(os.PathSeparator)) && clean != filepath.Clean(dir) {
		// also allow absolute path that exists if under home
		home, _ := os.UserHomeDir()
		if home == "" || !strings.HasPrefix(clean, home) {
			http.Error(w, "path not allowed", 403)
			return
		}
	}
	if st, err := os.Stat(clean); err != nil || st.IsDir() {
		http.Error(w, "not found", 404)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(clean))
	http.ServeFile(w, r, clean)
}
