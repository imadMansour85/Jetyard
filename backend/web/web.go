// Package web serves the statically exported frontend (frontend/out) from
// the backend, so the single Docker image needs one process and one port.
package web

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Handler serves files from dir. A clean URL like /streams is served from
// streams.html (Next.js static export layout), a directory from its
// index.html, and anything else gets 404.html with a 404 status.
func Handler(dir string) http.Handler {
	root := http.Dir(dir)
	files := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		p := path.Clean("/" + r.URL.Path)

		// Hashed build assets never change for a given name.
		if strings.HasPrefix(p, "/_next/static/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		switch {
		case p == "/":
			serveFile(w, r, root, "/index.html", http.StatusOK)
		case isFile(root, p):
			files.ServeHTTP(w, r)
		case isFile(root, p+".html"):
			serveFile(w, r, root, p+".html", http.StatusOK)
		case isFile(root, p+"/index.html"):
			serveFile(w, r, root, p+"/index.html", http.StatusOK)
		default:
			serveFile(w, r, root, "/404.html", http.StatusNotFound)
		}
	})
}

func isFile(root http.FileSystem, name string) bool {
	f, err := root.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	return err == nil && !info.IsDir()
}

// serveFile writes one file with an explicit status. http.ServeContent can't
// be used for the 404 page because it always answers 200.
func serveFile(w http.ResponseWriter, r *http.Request, root http.FileSystem, name string, status int) {
	f, err := root.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	if status == http.StatusOK {
		http.ServeContent(w, r, name, info.ModTime(), f)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		buf := make([]byte, 32*1024)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}
}

// Available reports whether dir looks like an exported frontend.
func Available(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !info.IsDir()
}
