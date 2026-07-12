package httpapi

import (
	"html/template"
	"io/fs"
	"time"
)

// ParseTemplate parses the index template from fsys with the app's template functions.
func ParseTemplate(fsys fs.FS) (*template.Template, error) {
	return template.New("index.html").
		Funcs(template.FuncMap{"formatDate": formatDate}).
		ParseFS(fsys, "template/index.html")
}

// formatDate renders a timestamp as "Month D, YYYY".
func formatDate(t time.Time) string {
	return t.Format("January 2, 2006")
}
