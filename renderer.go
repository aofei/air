package air

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// renderer is a renderer for rendering HTML templates.
type renderer struct {
	a         *Air
	loadOnce  *sync.Once
	loadError error
	watcher   *fsnotify.Watcher
	template  *template.Template
}

// newRenderer returns a new instance of the `renderer` with the a.
func newRenderer(a *Air) *renderer {
	return &renderer{
		a:        a,
		loadOnce: &sync.Once{},
	}
}

// load loads the stuff of the r up.
func (r *renderer) load() {
	defer func() {
		if r.loadError != nil {
			r.loadOnce = &sync.Once{}
		}
	}()

	if r.watcher == nil {
		r.watcher, r.loadError = fsnotify.NewWatcher()
		if r.loadError != nil {
			return
		}

		go func() {
			for {
				select {
				case <-r.watcher.Events:
					r.loadOnce = &sync.Once{}
				case err := <-r.watcher.Errors:
					r.a.errorLogger.Printf(
						"air: renderer watcher error: "+
							"%v",
						err,
					)
				}
			}
		}()
	}

	var tr string
	tr, r.loadError = filepath.Abs(r.a.RendererTemplateRoot)
	if r.loadError != nil {
		return
	}

	t := template.
		New("template").
		Delims(
			r.a.RendererTemplateLeftDelim,
			r.a.RendererTemplateRightDelim,
		).
		Funcs(template.FuncMap{
			"strlen":  strlen,
			"substr":  substr,
			"timefmt": timefmt,
			"locstr":  locstr,
		}).
		Funcs(r.a.RendererTemplateFuncMap)
	if r.loadError = filepath.Walk(
		tr,
		func(p string, fi os.FileInfo, err error) error {
			if fi == nil || fi.IsDir() || !stringSliceContainsCIly(
				r.a.RendererTemplateExts,
				filepath.Ext(p),
			) {
				return err
			}

			b, err := ioutil.ReadFile(p)
			if err != nil {
				return err
			}

			if _, err := t.New(filepath.ToSlash(
				p[len(tr)+1:],
			)).Parse(string(b)); err != nil {
				return err
			}

			return r.watcher.Add(p)
		},
	); r.loadError == nil {
		r.template = t
	}
}

// render renders the v into the w for the HTML template name.
func (r *renderer) render(
	w io.Writer,
	name string,
	v interface{},
	locstr func(string) string,
) error {
	if r.loadOnce.Do(r.load); r.loadError != nil {
		return r.loadError
	}

	t := r.template.Lookup(name)
	if t == nil {
		return fmt.Errorf("air: undefined html template: %s", name)
	}

	if !r.a.I18nEnabled {
		return t.Execute(w, v)
	}

	t, err := t.Clone()
	if err != nil {
		return err
	}

	return t.Funcs(template.FuncMap{
		"locstr": locstr,
	}).Execute(w, v)
}

// strlen returns the number of characters in the s.
func strlen(s string) int {
	return len([]rune(s))
}

// substr returns the substring consisting of the characters of the s starting
// at the index i and continuing up to, but not including, the character at the
// index j.
func substr(s string, i, j int) string {
	return string([]rune(s)[i:j])
}

// timefmt returns a textual representation of the t formatted for the layout.
func timefmt(t time.Time, layout string) string {
	return t.Format(layout)
}

// locstr returns the key without any changes.
func locstr(key string) string {
	return key
}
