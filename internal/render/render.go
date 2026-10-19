package render

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/kavalex/mysite/web"
)

// PageData is the data exposed to templates.
//
// No CSRF token field: in Go 1.25+ protection is implemented through
// http.CrossOriginProtection, so tokens in templates are not needed (see §6).
type PageData struct {
	Title     string
	Canonical string // populated by the handler as cfg.BaseURL + r.URL.Path
	Nonce     string // from middleware.CSPNonceFrom — for inline <script nonce="...">
	User      *UserData
	Message   *FlashMessage
	Data      any // arbitrary page data
}

type UserData struct {
	ID       int64
	Username string
	Email    string
}

type FlashMessage struct {
	Type string // "info", "error", "success"
	Text string
}

var pages map[string]*template.Template

func init() {
	funcs := template.FuncMap{
		"year":     func() int { return time.Now().Year() },
		"truncate": truncate,
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04")
		},
	}

	// Base template: layout + partials.
	base := template.Must(template.New("").Funcs(funcs).
		ParseFS(web.TemplatesFS,
			"templates/layouts/*.html",
			"templates/partials/*.html",
		))

	pageFiles, err := fs.Glob(web.TemplatesFS, "templates/pages/*.html")
	if err != nil {
		panic(err)
	}

	// Clone the base template per page. This is essential: without a clone,
	// {{ define "content" }} in pages/*.html would overwrite each other in a
	// single namespace.
	pages = make(map[string]*template.Template, len(pageFiles))
	for _, f := range pageFiles {
		name := strings.TrimSuffix(strings.TrimPrefix(f, "templates/pages/"), ".html")
		clone := template.Must(base.Clone())
		pages[name] = template.Must(clone.ParseFS(web.TemplatesFS, f))
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// Page renders page `name` (the file name without .html) into w.
func Page(w http.ResponseWriter, name string, data *PageData) error {
	tmpl, ok := pages[name]
	if !ok {
		return fmt.Errorf("render: unknown page %q", name)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, "base", data)
}

// PageWriter renders to an arbitrary io.Writer (used by tests).
func PageWriter(w io.Writer, name string, data *PageData) error {
	tmpl, ok := pages[name]
	if !ok {
		return fmt.Errorf("render: unknown page %q", name)
	}
	return tmpl.ExecuteTemplate(w, "base", data)
}

// StaticFS returns the sub-FS for static assets (without the "static/" prefix).
func StaticFS() fs.FS {
	sub, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
