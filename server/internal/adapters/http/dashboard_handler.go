package httpadapter

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

type DashboardHandler struct {
	files fs.FS
}

func NewDashboardHandler(files fs.FS) *DashboardHandler {
	return &DashboardHandler{files: files}
}

func (h *DashboardHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", h.serve)
}

func (h *DashboardHandler) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "api route not found"})
		return
	}

	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "." || name == "" {
		name = "index.html"
	}
	if !fs.ValidPath(name) {
		name = "index.html"
	}
	if _, err := fs.Stat(h.files, name); err != nil {
		name = "index.html"
	}

	// #nosec G703 -- name is normalized, validated as fs.ValidPath, and served from the configured dashboard fs.FS.
	http.ServeFileFS(w, r, h.files, name)
}
