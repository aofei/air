package air

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	"golang.org/x/text/language"
)

// i18n is a locale manager that adapts to the request's favorite conventions.
type i18n struct {
	locales map[string]map[string]string
	matcher language.Matcher
	watcher *fsnotify.Watcher
	once    *sync.Once
}

// theI18n is the singleton of the `i18n`.
var theI18n = &i18n{
	locales: map[string]map[string]string{},
	once:    &sync.Once{},
}

func init() {
	var err error
	if theI18n.watcher, err = fsnotify.NewWatcher(); err != nil {
		panic(fmt.Errorf("air: failed to build i18n watcher: %v", err))
	}

	go func() {
		for {
			select {
			case e := <-theI18n.watcher.Events:
				if I18nEnabled {
					DEBUG(
						"air: locale file event occurs",
						map[string]interface{}{
							"file":  e.Name,
							"event": e.Op.String(),
						},
					)
				}

				theI18n.once = &sync.Once{}
			case err := <-theI18n.watcher.Errors:
				if I18nEnabled {
					ERROR(
						"air: i18n watcher error",
						map[string]interface{}{
							"error": err.Error(),
						},
					)
				}
			}
		}
	}()
}

// localize localizes the r.
func (i *i18n) localize(r *Request) {
	if !I18nEnabled {
		return
	}

	i.once.Do(func() {
		lr, err := filepath.Abs(LocaleRoot)
		if err != nil {
			ERROR(
				"air: failed to get absolute representation "+
					"of locale root",
				map[string]interface{}{
					"error": err.Error(),
				},
			)

			return
		}

		lfns, err := filepath.Glob(filepath.Join(lr, "*.toml"))
		if err != nil {
			ERROR(
				"air: failed to get locale files",
				map[string]interface{}{
					"error": err.Error(),
				},
			)

			return
		}

		ls := make(map[string]map[string]string, len(lfns))
		ts := make([]language.Tag, 0, len(lfns))
		for _, lfn := range lfns {
			b, err := ioutil.ReadFile(lfn)
			if err != nil {
				ERROR(
					"air: failed to read locale file",
					map[string]interface{}{
						"error": err.Error(),
					},
				)

				return
			}

			l := map[string]string{}
			if err := toml.Unmarshal(b, &l); err != nil {
				ERROR(
					"air: failed to unmarshal locale file",
					map[string]interface{}{
						"error": err.Error(),
					},
				)

				return
			}

			t, err := language.Parse(strings.Replace(
				filepath.Base(lfn),
				".toml",
				"",
				1,
			))
			if err != nil {
				ERROR(
					"air: failed to parse locale",
					map[string]interface{}{
						"error": err.Error(),
					},
				)

				return
			}

			ls[t.String()] = l
			ts = append(ts, t)
		}

		i.locales = ls
		i.matcher = language.NewMatcher(ts)

		if err := i.watcher.Add(lr); err != nil {
			ERROR(
				"air: failed to watch locale files",
				map[string]interface{}{
					"error": err.Error(),
				},
			)
		}
	})

	als := []string{}
	if alh := r.Headers["accept-language"]; alh != nil {
		als = alh.Values
	}

	mt, _ := language.MatchStrings(i.matcher, als...)
	l := i.locales[mt.String()]

	r.localizedString = func(key string) string {
		if v, ok := l[key]; ok {
			return v
		} else if v, ok := i.locales[LocaleBase][key]; ok {
			return v
		}

		return key
	}
}
