package air

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// renderer is used to provide a way to render templates.
type renderer struct {
	air             *Air
	template        *template.Template
	templateFuncMap template.FuncMap
	watcher         *fsnotify.Watcher
}

// newRenderer returns a new instance of the `renderer`.
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

// init initializes the r.
func (r *renderer) init() error {
	if _, err := os.Stat(r.air.TemplateRoot); os.IsNotExist(err) {
		return nil
	}

	tr, err := filepath.Abs(r.air.TemplateRoot)
	if err != nil {
		return err
	}

	dirs, files, err := walkFiles(tr, r.air.TemplateExts)
	if err != nil {
		return err
	}

	if r.watcher == nil {
		if r.watcher, err = fsnotify.NewWatcher(); err != nil {
			return err
		}

		for _, d := range dirs {
			if err := r.watcher.Add(d); err != nil {
				return err
			}
		}

		go r.watchTemplates()
	}

	t := template.New("template")
	t.Funcs(r.templateFuncMap)
	t.Delims(r.air.TemplateLeftDelim, r.air.TemplateRightDelim)

	for _, f := range files {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}
		_, err = t.New(filepath.ToSlash(f[len(tr)+1:])).Parse(string(b))
		if err != nil {
			return err
		}
	}

	r.template = t

	return nil
}

// render renders the values into the w with the templateName.
func (r *renderer) render(
	w io.Writer,
	templateName string,
	values map[string]interface{},
) error {
	return r.template.ExecuteTemplate(w, templateName, values)
}

// watchTemplates watchs the changing of all template files.
func (r *renderer) watchTemplates() {
	for {
		select {
		case event := <-r.watcher.Events:
			r.air.Logger.Info(event)
			if event.Op == fsnotify.Create {
				r.watcher.Add(event.Name)
			}
			if err := r.init(); err != nil {
				r.air.Logger.Error(err)
			}
		case err := <-r.watcher.Errors:
			r.air.Logger.Error(err)
		}
	}
}

// walkFiles walks all files with the exts in all subdirs of the root
// recursively.
func walkFiles(
	root string,
	exts []string,
) (
	dirs []string,
	files []string,
	err error,
) {
	if err = filepath.Walk(
		root,
		func(path string, info os.FileInfo, err error) error {
			if info != nil && info.IsDir() {
				dirs = append(dirs, path)
			}
			return err
		},
	); err != nil {
		return nil, nil, err
	}

	for _, dir := range dirs {
		for _, ext := range exts {
			fs, err := filepath.Glob(filepath.Join(dir, "*"+ext))
			if err != nil {
				return nil, nil, err
			}
			files = append(files, fs...)
		}
	}

	return
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

// substr returns the substring consisting of the chars of the s starting at the
// index i and continuing up to, but not including, the char at the index j.
func substr(s string, i, j int) string {
	rs := []rune(s)
	return string(rs[i:j])
}

// timefmt returns a textual representation of the t formatted according to the
// layout.
func timefmt(t time.Time, layout string) string {
	return t.Format(layout)
}
