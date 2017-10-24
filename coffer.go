package air

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// coffer is used to provide a way to access binary asset files through the
// runtime memory.
type coffer struct {
	air     *Air
	assets  map[string]*Asset
	watcher *fsnotify.Watcher
}

// newCoffer returns a new instance of the `coffer`.
func newCoffer(a *Air) *coffer {
	return &coffer{
		air:    a,
		assets: map[string]*Asset{},
	}
}

// init initializes the c.
func (c *coffer) init() error {
	if !c.air.CofferEnabled {
		return nil
	} else if _, err := os.Stat(c.air.AssetRoot); os.IsNotExist(err) {
		return nil
	}

	ar, err := filepath.Abs(c.air.AssetRoot)
	if err != nil {
		return err
	}

	dirs, files, err := walkFiles(ar, c.air.AssetExts)
	if err != nil {
		return err
	}

	if c.watcher == nil {
		if c.watcher, err = fsnotify.NewWatcher(); err != nil {
			return err
		}

		for _, dir := range dirs {
			if err := c.watcher.Add(dir); err != nil {
				return err
			}
		}

		go c.watchAssets()
	}

	assets := map[string]*Asset{}

	for _, file := range files {
		fi, err := os.Stat(file)
		if err != nil {
			return err
		}

		b, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		if c.air.MinifierEnabled {
			if mt := mimeTypeByExt(filepath.Ext(file)); mt != "" {
				b, err = c.air.minifier.minify(mt, b)
				if err != nil {
					return err
				}
			}
		}

		assets[file] = &Asset{
			Name:    file,
			ModTime: fi.ModTime(),
			Reader:  bytes.NewReader(b),
		}
	}

	c.assets = assets

	return nil
}

// asset returns an `Asset` in the `Coffer` for the provided name.
//
// **Please use the `filepath.Abs()` to process the name before using.**
func (c *coffer) asset(name string) *Asset {
	return c.assets[name]
}

// watchTemplates watchs the changing of all asset files.
func (c *coffer) watchAssets() {
	for {
		select {
		case event := <-c.watcher.Events:
			c.air.Logger.INFO(event)

			if event.Op == fsnotify.Create {
				c.watcher.Add(event.Name)
			}

			if err := c.init(); err != nil {
				c.air.Logger.ERROR(err)
			}
		case err := <-c.watcher.Errors:
			c.air.Logger.ERROR(err)
		}
	}
}

// mimeTypeByExt returns a MIME type by the ext.
func mimeTypeByExt(ext string) string {
	switch ext {
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "text/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "text/xml"
	case ".svg":
		return "image/svg+xml"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	}
	return ""
}

// Asset is a binary asset file.
type Asset struct {
	Name    string
	ModTime time.Time
	Reader  *bytes.Reader
}
