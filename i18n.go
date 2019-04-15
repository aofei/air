package air

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	"golang.org/x/text/language"
)

// i18n is a locale manager that adapts to the request's favorite conventions.
type i18n struct {
	a         *Air
	loadOnce  *sync.Once
	loadError error
	watcher   *fsnotify.Watcher
	matcher   language.Matcher
	locales   map[string]map[string]string
}

// newI18n returns a new instance of the `i18n` with the a.
func newI18n(a *Air) *i18n {
	return &i18n{
		a:        a,
		loadOnce: &sync.Once{},
	}
}

// load loads the stuff of the i up.
func (i *i18n) load() {
	defer func() {
		if i.loadError != nil {
			i.loadOnce = &sync.Once{}
		}
	}()

	if i.watcher == nil {
		i.watcher, i.loadError = fsnotify.NewWatcher()
		if i.loadError != nil {
			return
		}

		go func() {
			for {
				select {
				case <-i.watcher.Events:
					i.loadOnce = &sync.Once{}
				case err := <-i.watcher.Errors:
					i.a.errorLogger.Printf(
						"air: i18n watcher error: %v",
						err,
					)
				}
			}
		}()
	}

	var lr string
	lr, i.loadError = filepath.Abs(i.a.I18nLocaleRoot)
	if i.loadError != nil {
		return
	}

	var fis []os.FileInfo
	if fis, i.loadError = ioutil.ReadDir(lr); i.loadError != nil {
		return
	}

	ts := make([]language.Tag, 0, len(fis))
	ls := make(map[string]map[string]string, len(ts))
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}

		var t language.Tag
		if ext := filepath.Ext(fi.Name()); strings.ToLower(
			ext,
		) != ".toml" {
			continue
		} else if t, i.loadError = language.Parse(strings.TrimSuffix(
			fi.Name(),
			ext,
		)); i.loadError != nil {
			return
		}

		n := filepath.Join(lr, fi.Name())
		l := map[string]string{}
		if _, i.loadError = toml.DecodeFile(n, &l); i.loadError != nil {
			return
		} else if i.loadError = i.watcher.Add(n); i.loadError != nil {
			return
		}

		ts = append(ts, t)
		ls[t.String()] = l
	}

	i.matcher = language.NewMatcher(ts)
	i.locales = ls
}

// localize localizes the r.
func (i *i18n) localize(r *Request) {
	if i.loadOnce.Do(i.load); i.loadError != nil {
		i.a.errorLogger.Printf(
			"air: failed to load i18n: %v",
			i.loadError,
		)

		r.localizedString = locstr

		return
	}

	t, _ := language.MatchStrings(i.matcher, r.Header["Accept-Language"]...)
	l := i.locales[t.String()]

	r.localizedString = func(key string) string {
		if v, ok := l[key]; ok {
			return v
		} else if l, ok := i.locales[i.a.I18nLocaleBase]; ok {
			if v, ok := l[key]; ok {
				return v
			}
		}

		return key
	}
}
