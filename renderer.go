package air

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
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
		minifier        *minify.M
		watcher         *fsnotify.Watcher
	}
)

// newRenderer returns a pointer of a new instance of the `renderer`.
func newRenderer(a *Air) *renderer {
	return &renderer{
		air:      a,
		template: template.New("template"),
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

	if _, err := os.Stat(c.TemplateRoot); err != nil && os.IsNotExist(err) {
		return nil
	}

	if c.TemplateMinified {
		r.minifier = minify.New()
		r.minifier.Add("text/html", &html.Minifier{
			KeepDefaultAttrVals: true,
			KeepDocumentTags:    true,
			KeepWhitespace:      true,
		})
	}

	if c.TemplateWatched {
		var err error
		if r.watcher, err = fsnotify.NewWatcher(); err != nil {
			return err
		}

		dirs, err := walkDirs(r.air.Config.TemplateRoot)
		if err != nil {
			return err
		}

		for _, dir := range dirs {
			if err := r.watcher.Add(dir); err != nil {
				return err
			}
		}

		go r.watchTemplates()
	}

	return r.parseTemplates()
}

// Render implements the `Renderer#Render()` by using the `template.Template`.
func (r *renderer) Render(w io.Writer, templateName string, data JSONMap) error {
	return r.template.ExecuteTemplate(w, templateName, data)
}

// parseTemplates parses all template files.
func (r *renderer) parseTemplates() error {
	c := r.air.Config

	tr := filepath.Clean(c.TemplateRoot)
	if _, err := os.Stat(tr); err != nil && os.IsNotExist(err) {
		return nil
	}

	dirs, err := walkDirs(tr)
	if err != nil {
		return err
	}

	var filenames []string
	for _, dir := range dirs {
		fns, err := filepath.Glob(fmt.Sprintf("%s/*%s", dir, c.TemplateExt))
		if err != nil {
			return err
		}
		filenames = append(filenames, fns...)
	}

	buf := &bytes.Buffer{}

	t := template.New("template")
	t.Funcs(r.templateFuncMap)
	t.Delims(c.TemplateLeftDelim, c.TemplateRightDelim)

	for _, filename := range filenames {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}

		if c.TemplateMinified {
			err := r.minifier.Minify("text/html", buf, bytes.NewReader(b))
			if err != nil {
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
		if _, err := t.New(name).Parse(string(b)); err != nil {
			return err
		}
	}

	r.template = t

	return nil
}

// watchTemplates watchs the changing of all template files.
func (r *renderer) watchTemplates() {
	for {
		select {
		case event := <-r.watcher.Events:
			r.air.Logger.Info(event)

			if event.Op == fsnotify.Create {
				s := event.String()
				s = s[:strings.Index(s, ":")]
				s = s[1 : len(s)-1]
				if filepath.Ext(s) != r.air.Config.TemplateExt {
					r.watcher.Add(s)
				}
			}

			if err := r.parseTemplates(); err != nil {
				r.air.Logger.Error(err)
			}
		case err := <-r.watcher.Errors:
			r.air.Logger.Error(err)
		}
	}
}

// walkDirs walks all subdirs of the root recursively.
func walkDirs(root string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			dirs = append(dirs, path)
		}
		return err
	})
	return dirs, err
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
