package view

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"sync"
	"syscall"
	"time"
)

const (
	// resultsBlock is the template block swapped into the page on htmx requests.
	resultsBlock = "results"
	staticDir    = "static"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// Renderer executes the page template and writes complete HTML responses.
type Renderer struct {
	tpl    *template.Template
	logger *slog.Logger
}

// NewRenderer builds a Renderer with the given template and logger.
func NewRenderer(tpl *template.Template, logger *slog.Logger) *Renderer {
	return &Renderer{tpl: tpl, logger: logger}
}

// ParseTemplate parses the index template with the app's template functions.
func ParseTemplate(tplFS, staticFS fs.FS) (*template.Template, error) {
	asset, err := assetURL(staticFS)
	if err != nil {
		return nil, err
	}
	return template.New("index.html").
		Funcs(template.FuncMap{"formatDate": formatDate, "asset": asset}).
		ParseFS(tplFS, "template/index.html")
}

// assetURL builds a helper that fingerprints static URLs so a changed file never comes from cache.
func assetURL(staticFS fs.FS) (func(string) (string, error), error) {
	entries, err := fs.ReadDir(staticFS, staticDir)
	if err != nil {
		return nil, err
	}
	urls := make(map[string]string, len(entries))
	for _, e := range entries {
		b, err := fs.ReadFile(staticFS, path.Join(staticDir, e.Name()))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		urls[e.Name()] = "/" + staticDir + "/" + e.Name() + "?v=" + hex.EncodeToString(sum[:4])
	}
	return func(name string) (string, error) {
		u, ok := urls[name]
		if !ok {
			return "", fmt.Errorf("unknown static asset %q", name)
		}
		return u, nil
	}, nil
}

// Render buffers the page for data and writes it with the given status.
// Partial renders only the results block.
func (v *Renderer) Render(w http.ResponseWriter, status int, data *SearchPage, partial bool) {
	buf, ok := bufPool.Get().(*bytes.Buffer)
	if !ok {
		buf = new(bytes.Buffer)
	}
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	name := v.tpl.Name()
	if partial {
		name = resultsBlock
	}
	if err := v.tpl.ExecuteTemplate(buf, name, data); err != nil {
		v.logger.Error("template execution error", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
			v.logger.Debug("connection aborted", slog.Any("error", err))
			return
		}
		v.logger.Error("error writing response", slog.Any("error", err))
	}
}

// Error renders the page with an error message so failures stay styled HTML.
func (v *Renderer) Error(w http.ResponseWriter, status int, msg string, partial bool) {
	v.Render(w, status, &SearchPage{Error: msg}, partial)
}

// formatDate renders a timestamp as "Month D, YYYY", empty for missing dates.
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("January 2, 2006")
}
