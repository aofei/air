package air

import (
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
	template *template.Template
	watcher  *fsnotify.Watcher
	once     *sync.Once
}

// rendererSingleton is the singleton of the `renderer`.
var rendererSingleton = &renderer{
	once: &sync.Once{},
}

func init() {
	var err error
	if rendererSingleton.watcher, err = fsnotify.NewWatcher(); err != nil {
		panic(err)
	}
	go func() {
		for {
			select {
			case event := <-rendererSingleton.watcher.Events:
				INFO(event)
				rendererSingleton.once = &sync.Once{}
			case err := <-rendererSingleton.watcher.Errors:
				ERROR(err)
			}
		}
	}()
}

// render renders the v into the w for the provided HTML template name.
func (r *renderer) render(w io.Writer, name string, v interface{}) error {
	r.once.Do(func() {
		tr, err := filepath.Abs(TemplateRoot)
		if err != nil {
			PANIC(err)
		}
		r.template = template.New("template").
			Delims(TemplateLeftDelim, TemplateRightDelim).
			Funcs(TemplateFuncMap)
		if err := filepath.Walk(
			tr,
			func(p string, fi os.FileInfo, err error) error {
				if fi == nil || !fi.IsDir() {
					return err
				}
				for _, e := range TemplateExts {
					fs, err := filepath.Glob(
						filepath.Join(p, "*"+e),
					)
					if err != nil {
						return err
					}
					for _, f := range fs {
						b, err := ioutil.ReadFile(f)
						if err != nil {
							return err
						}
						if _, err := r.template.New(
							filepath.ToSlash(
								f[len(tr)+1:],
							),
						).Parse(string(b)); err != nil {
							return err
						}
					}
				}
				return r.watcher.Add(p)
			},
		); err != nil {
			PANIC(err)
		}
	})
	return r.template.ExecuteTemplate(w, name, v)
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

// timefmt returns a textual representation of the t formatted for the provided
// layout.
func timefmt(t time.Time, layout string) string {
	return t.Format(layout)
}
