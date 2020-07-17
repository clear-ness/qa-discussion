package utils

import (
	"bytes"
	"errors"
	"html/template"
	"io"

	"github.com/clear-ness/qa-discussion/mlog"
)

type HTMLTemplate struct {
	Templates    *template.Template
	TemplateName string
	Props        map[string]interface{}
	Html         map[string]template.HTML
}

func NewHTMLTemplate(templates *template.Template, templateName string) *HTMLTemplate {
	return &HTMLTemplate{
		Templates:    templates,
		TemplateName: templateName,
		Props:        make(map[string]interface{}),
		Html:         make(map[string]template.HTML),
	}
}

func (t *HTMLTemplate) Render() string {
	var text bytes.Buffer
	t.RenderToWriter(&text)
	return text.String()
}

func (t *HTMLTemplate) RenderToWriter(w io.Writer) error {
	if t.Templates == nil {
		return errors.New("no html templates")
	}

	if err := t.Templates.ExecuteTemplate(w, t.TemplateName, t); err != nil {
		mlog.Error("Error rendering template", mlog.String("template_name", t.TemplateName), mlog.Err(err))
		return err
	}

	return nil
}
