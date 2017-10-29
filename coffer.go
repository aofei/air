package air

import (
	"bytes"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// coffer is a binary asset file manager that uses runtime memory to reduce disk
// I/O pressure.
type coffer struct {
	assets  map[string]*asset
	watcher *fsnotify.Watcher
}

// cofferSingleton is the singleton of the `coffer`.
var cofferSingleton = &coffer{
	assets: map[string]*asset{},
}

// asset returns an `asset` from the `coffer` for the provided name.
func (c *coffer) asset(name string) (*asset, error) {
	if !CofferEnabled {
		return nil, nil
	}

	if !filepath.IsAbs(name) {
		var err error
		if name, err = filepath.Abs(name); err != nil {
			return nil, err
		}
	} else if a, ok := c.assets[name]; ok {
		return a, nil
	}

	if _, err := os.Stat(AssetRoot); os.IsNotExist(err) {
		return nil, nil
	}

	ar, err := filepath.Abs(AssetRoot)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(name, ar) {
		return nil, nil
	}

	ext := strings.ToLower(filepath.Ext(name))
	isAsset := false
	for _, ae := range AssetExts {
		if strings.ToLower(ae) == ext {
			isAsset = true
		}
	}
	if !isAsset {
		return nil, nil
	}

	fi, err := os.Stat(name)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	if mt := mime.TypeByExtension(ext); mt != "" {
		if b, err = minifierSingleton.minify(mt, b); err != nil {
			return nil, err
		}
	}

	if c.watcher == nil {
		if c.watcher, err = fsnotify.NewWatcher(); err != nil {
			return nil, err
		}
		go func() {
			for {
				select {
				case event := <-c.watcher.Events:
					if CofferEnabled {
						INFO(event)
					}
					delete(c.assets, event.Name)
				case err := <-c.watcher.Errors:
					if CofferEnabled {
						ERROR(err)
					}
				}
			}
		}()
	} else if err := c.watcher.Add(name); err != nil {
		return nil, err
	}

	c.assets[name] = &asset{
		name:    name,
		modTime: fi.ModTime(),
		reader:  bytes.NewReader(b),
	}

	return c.assets[name], nil
}

// asset is a binary asset file.
type asset struct {
	name    string
	modTime time.Time
	reader  *bytes.Reader
}
