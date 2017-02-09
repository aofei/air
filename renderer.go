package air

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

type (
	// Renderer is used to provide a `Render()` method for an `Air` instance for renders a
	// "text/html" HTTP response.
	Renderer interface {
		// SetTemplateFunc sets the func f into template func map with the name.
		SetTemplateFunc(name string, f interface{})

		// ParseTemplates parses template files. It will be called in the `Air#Serve()`.
		ParseTemplates() error

		// Render renders the data into the w with the templateName.
		Render(w io.Writer, templateName string, data JSONMap) error
	}

	// renderer implements the `Renderer` by using the `template.Template`.
	renderer struct {
		air *Air

		template        *template.Template
		templateFuncMap template.FuncMap
	}
)

// newRenderer returns a pointer of a new instance of the `renderer`.
func newRenderer(a *Air) *renderer {
	return &renderer{
		air: a,
		templateFuncMap: template.FuncMap{
			"strlen":  strlen,
			"strcat":  strcat,
			"substr":  substr,
			"timefmt": timefmt,
		},
	}
}

// SetTemplateFunc implements the `Renderer#SetTemplateFunc()` by using the `template.Template`.
func (r *renderer) SetTemplateFunc(name string, f interface{}) {
	r.templateFuncMap[name] = f
}

// ParseTemplates implements the `Renderer#ParseTemplates()` by using the `template.Template`.
//
// e.g. r.air.Config.TemplateRoot == "templates" && r.air.Config.TemplateExt == ".html"
//
// templates/
//   index.html
//   login.html
//   register.html
//
// templates/parts/
//   header.html
//   footer.html
//
// will be parsed into:
//
// "index.html", "login.html", "register.html", "parts/header.html", "parts/footer.html".
func (r *renderer) ParseTemplates() error {
	c := r.air.Config

	tr := filepath.Clean(c.TemplateRoot)
	if _, err := os.Stat(tr); err != nil && os.IsNotExist(err) {
		return nil
	}

	var filenames []string
	err := filepath.Walk(tr, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return err
		}
		fns, err := filepath.Glob(path + "/*" + c.TemplateExt)
		filenames = append(filenames, fns...)
		return err
	})
	if err != nil {
		return err
	}

	m := minify.New()
	m.Add("text/html", &html.Minifier{
		KeepDefaultAttrVals: true,
		KeepDocumentTags:    true,
	})
	buf := &bytes.Buffer{}

	for _, filename := range filenames {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}

		if c.MinifyTemplate {
			if err := m.Minify("text/html", buf, bytes.NewReader(b)); err != nil {
				return err
			}
			b = buf.Bytes()
			buf.Reset()
		}

		start := 0
		if tr != "." {
			start = len(tr) + 1
		}

		name := filepath.ToSlash(filename[start:])

		if r.template == nil {
			r.template = template.New(name)
			r.template.Funcs(r.templateFuncMap)
			r.template.Delims(c.TemplateLeftDelim, c.TemplateRightDelim)
		}

		if _, err := r.template.New(name).Parse(string(b)); err != nil {
			return err
		}
	}

	return nil
}

// Render implements the `Renderer#Render()` by using the `template.Template`.
func (r *renderer) Render(w io.Writer, templateName string, data JSONMap) error {
	return r.template.ExecuteTemplate(w, templateName, data)
}

// strlen returns the number of chars in the s.
func strlen(s string) int {
	return len([]rune(s))
}

// strcat returns a string that is catenated to the tail of the s by the ss.
func strcat(s string, ss ...string) string {
	for i := range ss {
		s = fmt.Sprintf("%s%s", s, ss[i])
	}
	return s
}

// substr returns the substring consisting of the chars of the s starting at the index i and
// continuing up to, but not including, the char at the index j.
func substr(s string, i, j int) string {
	rs := []rune(s)
	return string(rs[i:j])
}

// timefmt returns a textual representation of the t formatted according to the layout.
func timefmt(t time.Time, layout string) string {
	return t.Format(layout)
}
