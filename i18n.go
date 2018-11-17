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
	a       *Air
	locales map[string]map[string]string
	matcher language.Matcher
	watcher *fsnotify.Watcher
	once    *sync.Once
}

// newI18n returns a new instance of the `i18n` with the a.
func newI18n(a *Air) *i18n {
	i := &i18n{
		a:       a,
		locales: map[string]map[string]string{},
		once:    &sync.Once{},
	}

	var err error
	if i.watcher, err = fsnotify.NewWatcher(); err != nil {
		panic(fmt.Errorf("air: failed to build i18n watcher: %v", err))
	}

	go func() {
		for {
			select {
			case e := <-i.watcher.Events:
				if a.I18nEnabled {
					a.DEBUG(
						"air: locale file event occurs",
						map[string]interface{}{
							"file":  e.Name,
							"event": e.Op.String(),
						},
					)
				}

				i.once = &sync.Once{}
			case err := <-i.watcher.Errors:
				if a.I18nEnabled {
					a.ERROR(
						"air: i18n watcher error",
						map[string]interface{}{
							"error": err.Error(),
						},
					)
				}
			}
		}
	}()

	return i
}

// localize localizes the r.
func (i *i18n) localize(r *Request) {
	if !i.a.I18nEnabled {
		r.localizedString = func(key string) string {
			return key
		}

		return
	}

	i.once.Do(func() {
		lr, err := filepath.Abs(i.a.LocaleRoot)
		if err != nil {
			i.a.ERROR(
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
			i.a.ERROR(
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
				i.a.ERROR(
					"air: failed to read locale file",
					map[string]interface{}{
						"error": err.Error(),
					},
				)

				return
			}

			l := map[string]string{}
			if err := toml.Unmarshal(b, &l); err != nil {
				i.a.ERROR(
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
				i.a.ERROR(
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
			i.a.ERROR(
				"air: failed to watch locale files",
				map[string]interface{}{
					"error": err.Error(),
				},
			)
		}
	})

	t, _ := language.MatchStrings(i.matcher, r.Header["Accept-Language"]...)
	l := i.locales[t.String()]

	r.localizedString = func(key string) string {
		if v, ok := l[key]; ok {
			return v
		} else if v, ok := i.locales[i.a.LocaleBase][key]; ok {
			return v
		}

		return key
	}
}
