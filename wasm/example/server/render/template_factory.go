package render

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
)

type ExampleRoute int

const (
	IndexRoute ExampleRoute = iota
	DefaultRoute
)

//go:embed templates/*
var FS embed.FS

type HTMLTemplate struct {
	inner *template.Template
	funcs template.FuncMap
}

func (t *HTMLTemplate) SetFuncs(funcs template.FuncMap) {
	t.funcs = funcs
}

func (t *HTMLTemplate) Execute(w io.Writer, data interface{}) error {
	return t.inner.Funcs(t.funcs).Execute(w, data)
}

func GetTemplate(route ExampleRoute) *HTMLTemplate {
	var tmpl *HTMLTemplate
	switch route {
	case IndexRoute:
		tmpl = newTemplate("templates/index.html", "templates/common.html")
	case DefaultRoute:
		tmpl = newTemplate("templates/404.html")
	}
	return tmpl
}

func newTemplate(embedded ...string) *HTMLTemplate {
	var parsed *template.Template
	var err error

	if len(embedded) > 0 {
		parsed, err = template.ParseFS(FS, embedded...)
		if err != nil {
			log.Printf("warning: some child templates could not be parsed: %v", fmt.Errorf("failed to parse child templates: %w", err))
		}
	}

	return &HTMLTemplate{inner: parsed}
}
