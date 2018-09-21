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
	template *template.Template
	watcher  *fsnotify.Watcher
	once     *sync.Once
}

// theRenderer is the singleton of the `renderer`.
var theRenderer = &renderer{
	once: &sync.Once{},
}

func init() {
	var err error
	if theRenderer.watcher, err = fsnotify.NewWatcher(); err != nil {
		panic(fmt.Errorf(
			"air: failed to build renderer watcher: %v",
			err,
		))
	}

	go func() {
		for {
			select {
			case e := <-theRenderer.watcher.Events:
				DEBUG(
					"air: template file event occurs",
					map[string]interface{}{
						"file":  e.Name,
						"event": e.Op.String(),
					},
				)
				theRenderer.once = &sync.Once{}
			case err := <-theRenderer.watcher.Errors:
				ERROR(
					"air: renderer watcher error",
					map[string]interface{}{
						"error": err.Error(),
					},
				)
			}
		}
	}()
}

// render renders the v into the w for the HTML template name.
func (r *renderer) render(
	w io.Writer,
	name string,
	v interface{},
	locstr func(string) string,
) error {
	r.once.Do(func() {
		tr, err := filepath.Abs(TemplateRoot)
		if err != nil {
			ERROR(
				"air: failed to get absolute representation "+
					"of template root",
				map[string]interface{}{
					"error": err.Error(),
				},
			)

			return
		}

		r.template = template.
			New("template").
			Delims(TemplateLeftDelim, TemplateRightDelim).
			Funcs(template.FuncMap{
				"locstr": func(key string) string {
					return key
				},
			}).
			Funcs(TemplateFuncMap)
		if err := filepath.Walk(
			tr,
			func(p string, fi os.FileInfo, err error) error {
				if fi == nil || !fi.IsDir() {
					return err
				}

				for _, e := range TemplateExts {
					fns, err := filepath.Glob(
						filepath.Join(p, "*"+e),
					)
					if err != nil {
						return err
					}

					for _, fn := range fns {
						b, err := ioutil.ReadFile(fn)
						if err != nil {
							return err
						}

						if _, err := r.template.New(
							filepath.ToSlash(
								fn[len(tr)+1:],
							),
						).Parse(string(b)); err != nil {
							return err
						}
					}
				}

				return r.watcher.Add(p)
			},
		); err != nil {
			ERROR(
				"air: failed to walk template files",
				map[string]interface{}{
					"error": err.Error(),
				},
			)
		}
	})

	if locstr != nil {
		t, err := r.template.Clone()
		if err != nil {
			return err
		}

		return t.Funcs(template.FuncMap{
			"locstr": locstr,
		}).ExecuteTemplate(w, name, v)
	}

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

// timefmt returns a textual representation of the t formatted for the layout.
func timefmt(t time.Time, layout string) string {
	return t.Format(layout)
}
