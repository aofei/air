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
	a        *Air
	loadOnce *sync.Once
	watcher  *fsnotify.Watcher
	locales  map[string]map[string]string
	matcher  language.Matcher
}

// newI18n returns a new instance of the `i18n` with the a.
func newI18n(a *Air) *i18n {
	return &i18n{
		a:        a,
		loadOnce: &sync.Once{},
		locales:  map[string]map[string]string{},
		matcher:  language.NewMatcher(nil),
	}
}

// load loads the stuff of the i up.
func (i *i18n) load() error {
	if i.watcher == nil {
		var err error
		if i.watcher, err = fsnotify.NewWatcher(); err != nil {
			return err
		}

		go func() {
			for {
				select {
				case e := <-i.watcher.Events:
					i.a.DEBUG(
						"air: locale file event occurs",
						map[string]interface{}{
							"file":  e.Name,
							"event": e.Op.String(),
						},
					)

					i.loadOnce = &sync.Once{}
				case err := <-i.watcher.Errors:
					i.a.ERROR(
						"air: i18n watcher error",
						map[string]interface{}{
							"error": err.Error(),
						},
					)
				}
			}
		}()
	}

	lr, err := filepath.Abs(i.a.LocaleRoot)
	if err != nil {
		return err
	}

	if err := i.watcher.Add(lr); err != nil {
		return err
	}

	lfns, err := filepath.Glob(filepath.Join(lr, "*.toml"))
	if err != nil {
		return err
	}

	ls := make(map[string]map[string]string, len(lfns))
	ts := make([]language.Tag, 0, len(lfns))
	for _, lfn := range lfns {
		b, err := ioutil.ReadFile(lfn)
		if err != nil {
			return err
		}

		l := map[string]string{}
		if err := toml.Unmarshal(b, &l); err != nil {
			return err
		}

		t, err := language.Parse(strings.Replace(
			filepath.Base(lfn),
			".toml",
			"",
			1,
		))
		if err != nil {
			return err
		}

		ls[t.String()] = l
		ts = append(ts, t)
	}

	i.locales = ls
	i.matcher = language.NewMatcher(ts)

	return nil
}

// localize localizes the r.
func (i *i18n) localize(r *Request) {
	var err error
	i.loadOnce.Do(func() {
		err = i.load()
	})
	if err != nil {
		i.a.ERROR("air: failed to load i18n", map[string]interface{}{
			"error": err.Error(),
		})

		i.loadOnce = &sync.Once{}
	}

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
