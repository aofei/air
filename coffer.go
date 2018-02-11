package air

import (
	"io/ioutil"
	"mime"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// coffer is a binary asset file manager that uses runtime memory to reduce disk
// I/O pressure.
type coffer struct {
	assets  map[string]*asset
	watcher *fsnotify.Watcher
}

// theCoffer is the singleton of the `coffer`.
var theCoffer = &coffer{
	assets: map[string]*asset{},
}

func init() {
	var err error
	if theCoffer.watcher, err = fsnotify.NewWatcher(); err != nil {
		panic(err)
	}
	go func() {
		for {
			select {
			case event := <-theCoffer.watcher.Events:
				if CofferEnabled {
					INFO(event)
				}
				delete(theCoffer.assets, event.Name)
			case err := <-theCoffer.watcher.Errors:
				if CofferEnabled {
					ERROR(err)
				}
			}
		}
	}()
}

// asset returns an `asset` from the `coffer` for the provided name.
func (c *coffer) asset(name string) (*asset, error) {
	if !CofferEnabled {
		return nil, nil
	}

	if a, ok := c.assets[name]; ok {
		return a, nil
	} else if ar, err := filepath.Abs(AssetRoot); err != nil {
		return nil, err
	} else if !strings.HasPrefix(name, ar) {
		return nil, nil
	}

	ext := strings.ToLower(filepath.Ext(name))
	for i := range AssetExts {
		if strings.ToLower(AssetExts[i]) == ext {
			break
		} else if i == len(AssetExts)-1 {
			return nil, nil
		}
	}

	b, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	if mt := mime.TypeByExtension(ext); mt != "" {
		if b, err = theMinifier.minify(mt, b); err != nil {
			return nil, err
		}
	}

	if err := c.watcher.Add(name); err != nil {
		return nil, err
	}

	c.assets[name] = &asset{
		name:    name,
		content: b,
	}

	return c.assets[name], nil
}

// asset is a binary asset file.
type asset struct {
	name    string
	content []byte
}
