package air

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// coffer is used to provide an `asset()` method for an `Air` instance
// for accesses binary asset files by using the runtime memory.
type coffer struct {
	air *Air

	assets  map[string]*Asset
	watcher *fsnotify.Watcher
}

// newCoffer returns a pointer of a new instance of the `coffer`.
func newCoffer(a *Air) *coffer {
	return &coffer{
		air:    a,
		assets: map[string]*Asset{},
	}
}

// init initializes the `Coffer`. It will be called in the `Air#Serve()`.
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

		assets[file] = NewAsset(file, fi.ModTime(), b)
	}

	c.assets = assets

	return nil
}

// asset returns an `Asset` in the `Coffer` for the provided name.
//
// **Please use the `filepath.Abs()` to process the name before calling.**
func (c *coffer) asset(name string) *Asset {
	return c.assets[name]
}

// watchTemplates watchs the changing of all asset files.
func (c *coffer) watchAssets() {
	for {
		select {
		case event := <-c.watcher.Events:
			c.air.Logger.Info(event)

			if event.Op == fsnotify.Create {
				c.watcher.Add(event.Name)
			}

			if err := c.init(); err != nil {
				c.air.Logger.Error(err)
			}
		case err := <-c.watcher.Errors:
			c.air.Logger.Error(err)
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
	}
	return ""
}

// Asset is a binary asset file.
type Asset struct {
	name    string
	modTime time.Time
	reader  *bytes.Reader
}

// NewAsset returns a pointer of a new instance of the `Asset`.
func NewAsset(name string, modTime time.Time, content []byte) *Asset {
	return &Asset{
		name:    name,
		modTime: modTime,
		reader:  bytes.NewReader(content),
	}
}

// Name returns the name of the a.
func (a *Asset) Name() string {
	return a.name
}

// ModTime returns the modTime of the a.
func (a *Asset) ModTime() time.Time {
	return a.modTime
}

// Read implements the `io.Reader`.
func (a *Asset) Read(b []byte) (int, error) {
	return a.reader.Read(b)
}

// Seek implements the `io.Seeker`.
func (a *Asset) Seek(offset int64, whence int) (int64, error) {
	return a.reader.Seek(offset, whence)
}
