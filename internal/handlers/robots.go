package handlers

import (
	"io/fs"
	"net/http"

	"github.com/kavalex/mysite/internal/render"
)

func (h *Handler) Robots(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(render.StaticFS(), "robots.txt")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}
