package air

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// renderer is used to render the HTML templates.
type renderer struct {
	template *template.Template
	watcher  *fsnotify.Watcher
}

// rendererSingleton is the singleton instance of the `renderer`.
var rendererSingleton = &renderer{}

// render renders the values into the w with the templateName.
func (r *renderer) render(
	w io.Writer,
	templateName string,
	values map[string]interface{},
) error {
	if r.template == nil {
		r.template.New("template").
			Delims(TemplateLeftDelim, TemplateRightDelim).
			Funcs(TemplateFuncMap)
	} else if t := r.template.Lookup(templateName); t != nil {
		return t.Execute(w, values)
	}

	tr, err := filepath.Abs(TemplateRoot)
	if err != nil {
		return err
	}

	tn := filepath.Join(tr, templateName)
	if _, err := os.Stat(tn); os.IsNotExist(err) {
		return err
	}

	ext := strings.ToLower(filepath.Ext(tn))
	isTemplate := false
	for _, te := range TemplateExts {
		if strings.ToLower(te) == ext {
			isTemplate = true
		}
	}
	if !isTemplate {
		return nil
	}

	if r.watcher == nil {
		if r.watcher, err = fsnotify.NewWatcher(); err != nil {
			return err
		}

		go func() {
			for {
				select {
				case event := <-r.watcher.Events:
					INFO(event)
					r.template = nil
				case err := <-r.watcher.Errors:
					ERROR(err)
				}
			}
		}()
	} else if err := r.watcher.Add(tn); err != nil {
		return err
	}

	t, err := r.template.New(templateName).ParseFiles(tn)
	if err != nil {
		return err
	}

	return t.Execute(w, values)
}

// strlen returns the number of characters in the s.
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
	return string([]rune(s)[i:j])
}

// timefmt returns a textual representation of the t formatted according to the
// layout.
func timefmt(t time.Time, layout string) string {
	return t.Format(layout)
}
