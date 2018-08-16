package air

import (
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
		panic(err)
	}
	go func() {
		for {
			select {
			case event := <-theI18n.watcher.Events:
				if I18nEnabled {
					INFO(event.String())
				}
				theI18n.once = &sync.Once{}
			case err := <-theI18n.watcher.Errors:
				if I18nEnabled {
					ERROR(err.Error())
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
			PANIC(err.Error())
		}

		lfns, err := filepath.Glob(filepath.Join(lr, "*.toml"))
		if err != nil {
			PANIC(err.Error())
		}

		ls := make(map[string]map[string]string, len(lfns))
		ts := make([]language.Tag, 0, len(lfns))
		for _, lfn := range lfns {
			b, err := ioutil.ReadFile(lfn)
			if err != nil {
				PANIC(err.Error())
			}

			l := map[string]string{}
			if err := toml.Unmarshal(b, &l); err != nil {
				PANIC(err.Error())
			}

			t, err := language.Parse(strings.Replace(
				filepath.Base(lfn),
				".toml",
				"",
				1,
			))
			if err != nil {
				PANIC(err.Error())
			}

			ls[t.String()] = l
			ts = append(ts, t)
		}

		i.locales = ls
		i.matcher = language.NewMatcher(ts)
		i.watcher.Add(lr)
	})

	mt, _ := language.MatchStrings(i.matcher, r.Headers["Accept-Language"])
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
