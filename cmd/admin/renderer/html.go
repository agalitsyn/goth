package renderer

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/agalitsyn/goth/internal/auth"
	"github.com/agalitsyn/goth/internal/model"
	"github.com/agalitsyn/goth/pkg/httptools"
)

const (
	SmartBlock   = ""
	BaseBlock    = "base"
	ContentBlock = "content"
	ErrorBlock   = "error"
)

type pageData struct {
	// data for layout
	User *model.User
	Path string

	// data for block
	Data any
}

type errorData struct {
	Message template.HTML
	Error   string
}

func newPageData(r *http.Request) pageData {
	data := pageData{Path: r.URL.Path}
	user, err := auth.UserFromContext(r.Context())
	if err == nil {
		data.User = user
	}
	return data
}

type HTMLRenderer struct {
	Debug            bool
	templateRenderer *httptools.TemplateRenderer
}

func NewHTMLRenderer(renderer *httptools.TemplateRenderer) *HTMLRenderer {
	return &HTMLRenderer{
		templateRenderer: renderer,
	}
}

func (c *HTMLRenderer) Render(w http.ResponseWriter, r *http.Request, status int, template, block string, data any) {
	logAttrs := []any{"template", template, "block", block}
	if block != SmartBlock && isHTMXRequest(r) {
		slog.Debug("render html", logAttrs...)
		c.templateRenderer.Render(w, status, template, block, data)
		return
	}

	if block == SmartBlock {
		block = BaseBlock
		if isHTMXRequest(r) {
			block = ContentBlock
		}
	}

	pd := newPageData(r)
	pd.Data = data

	logAttrs = append(logAttrs, "path", pd.Path, "authenticated", pd.User != nil)
	slog.Debug("render html", logAttrs...)
	c.templateRenderer.Render(w, status, template, block, pd)
}

func (c *HTMLRenderer) Error(w http.ResponseWriter, r *http.Request, status int, msg string, err error) {
	data := errorData{
		Message: template.HTML(msg),
	}
	if c.Debug && err != nil {
		data.Error = err.Error()
	}

	// Any HTMX errors are rendered as a static block on defined in template page region
	if isHTMXRequest(r) {
		w.Header().Add("HX-Retarget", "#general-error")
		w.Header().Add("HX-Reswap", "innerHTML")
		c.Render(w, r, http.StatusOK, "error.tmpl.html", ErrorBlock, data)
		return
	}

	if status == http.StatusNotFound {
		c.Render(w, r, status, "404.tmpl.html", BaseBlock, data)
		return
	}

	c.Render(w, r, status, "500.tmpl.html", BaseBlock, data)
}

func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
