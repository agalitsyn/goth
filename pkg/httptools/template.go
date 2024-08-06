package httptools

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
)

type TemplateRenderer struct {
	cache map[string]*template.Template
}

func NewTemplateRenderer(templatesCache map[string]*template.Template) *TemplateRenderer {
	return &TemplateRenderer{
		cache: templatesCache,
	}
}

func (s *TemplateRenderer) Render(w http.ResponseWriter, status int, template, block string, data any) {
	ts, ok := s.cache[template]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", template)
		slog.Error("could not fetch template", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	err := ts.ExecuteTemplate(buf, block, data)
	if err != nil {
		slog.Error("could not execute template", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, err = buf.WriteTo(w)
	if err != nil {
		slog.Error("could not write template content", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func NewTemplateCache(
	embedFiles embed.FS,
	templatesFolder string,
	funcs template.FuncMap,
) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	root, err := fs.Glob(embedFiles, templatesFolder+"/*.tmpl.html")
	if err != nil {
		return nil, err
	}
	tree, err := fs.Glob(embedFiles, templatesFolder+"/*/*.tmpl.html")
	if err != nil {
		return nil, err
	}
	pages := append(root, tree...)

	for _, page := range pages {
		name := filepath.Base(page)
		patterns := []string{
			templatesFolder + "/*.tmpl.html",
			templatesFolder + "/partials/*.tmpl.html",
			page,
		}

		ts, err := template.New(name).Funcs(funcs).ParseFS(embedFiles, patterns...)
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}

	return cache, nil
}
