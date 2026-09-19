package mail

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"time"
)

//go:embed email-templates/*.html email-templates/partials/*.html
var emailTemplateFS embed.FS

type ShellData struct {
	Title   string
	Heading string
	Year    int
}

func NewShellData(title, heading string) ShellData {
	return ShellData{
		Title:   title,
		Heading: heading,
		Year:    time.Now().Year(),
	}
}

var parsedTemplates *template.Template

func templates() (*template.Template, error) {
	if parsedTemplates != nil {
		return parsedTemplates, nil
	}
	tmpl, err := template.ParseFS(
		emailTemplateFS,
		"email-templates/*.html",
		"email-templates/partials/*.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse email templates: %w", err)
	}
	parsedTemplates = tmpl
	return parsedTemplates, nil
}

func Render(name string, data any) (string, error) {
	tmpl, err := templates()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("render template %q: %w", name, err)
	}
	return buf.String(), nil
}
