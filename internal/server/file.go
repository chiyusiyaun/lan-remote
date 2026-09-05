package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	// optional dest dir (any path)
	if dest := r.URL.Query().Get("dest"); dest != "" {
		clean := filepath.Clean(dest)
		if allowedPath(clean, "") {
			if st, err := os.Stat(clean); err == nil && st.IsDir() {
				dir = clean
			}
		}
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
	home, _ := os.UserHomeDir()
	clean := filepath.Clean(path)
	if !allowedPath(clean, home) {
		http.Error(w, "path not allowed", 403)
		return
	}
	if st, err := os.Stat(clean); err != nil || st.IsDir() {
		http.Error(w, "not found", 404)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(clean))
	http.ServeFile(w, r, clean)
}

// handleListFiles lists a remote directory (PIN required).
// Query: path (optional, default = receive dir). Includes subdirectories.
func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	pin := r.Header.Get("X-LR-Pin")
	if pin == "" {
		pin = r.URL.Query().Get("pin")
	}
	if s.PIN() == "" || pin != s.PIN() {
		http.Error(w, "bad pin", 403)
		return
	}

	home, _ := os.UserHomeDir()
	root := r.URL.Query().Get("path")

	// Special: list drives / filesystem roots
	if root == "roots" || root == "/" && runtime.GOOS == "windows" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    true,
			"dir":   "roots",
			"parent": "",
			"home":  home,
			"is_roots": true,
			"files": listRoots(),
		})
		return
	}

	if root == "" {
		d, err := receiveDir()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		root = d
	}

	clean := filepath.Clean(root)
	if !allowedPath(clean, home) {
		http.Error(w, "path not allowed", 403)
		return
	}
	st, err := os.Stat(clean)
	if err != nil || !st.IsDir() {
		http.Error(w, "not a directory", 400)
		return
	}

	ents, err := os.ReadDir(clean)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	type item struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		Size  int64  `json:"size"`
		Mod   string `json:"mod"`
		IsDir bool   `json:"is_dir"`
	}
	items := make([]item, 0, len(ents))
	for _, e := range ents {
		info, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, item{
			Name:  e.Name(),
			Path:  filepath.Join(clean, e.Name()),
			Size:  info.Size(),
			Mod:   info.ModTime().Format("2006-01-02 15:04"),
			IsDir: e.IsDir(),
		})
	}
	parent := filepath.Dir(clean)
	if parent == clean {
		parent = ""
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"dir":    clean,
		"parent": parent,
		"home":   home,
		"files":  items,
	})
}

func allowedPath(p, home string) bool {
	// No filesystem jail: PIN is the only gate. Any path the OS can access is allowed.
	return p != ""
}

func listRoots() []map[string]interface{} {
	var out []map[string]interface{}
	if runtime.GOOS == "windows" {
		for _, letter := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
			root := string(letter) + `:\`
			if st, err := os.Stat(root); err == nil && st.IsDir() {
				out = append(out, map[string]interface{}{
					"name":  string(letter) + ":",
					"path":  root,
					"is_dir": true,
					"size":  int64(0),
					"mod":   "",
				})
			}
		}
	} else {
		out = append(out, map[string]interface{}{
			"name":  "/",
			"path":  "/",
			"is_dir": true,
			"size":  int64(0),
			"mod":   "",
		})
	}
	return out
}
