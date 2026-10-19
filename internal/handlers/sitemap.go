package handlers

import (
	"encoding/xml"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kavalex/mysite/internal/middleware"
)

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(h.cfg.BaseURL, "/")
	entries := []sitemapURL{
		{Loc: base + "/"},
		{Loc: base + "/about"},
	}
	// When articles/posts appear:
	//   rows, _ := h.store.DB().QueryContext(r.Context(),
	//       `SELECT slug, updated_at FROM posts WHERE published = 1`)
	//   defer rows.Close()
	//   for rows.Next() { ... entries = append(entries, ...) }

	set := sitemapSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  entries,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(set); err != nil {
		slog.Error("sitemap encode failed",
			"err", err,
			"request_id", middleware.RequestIDFrom(r.Context()))
	}
}
