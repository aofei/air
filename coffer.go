package air

import (
	"bytes"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// coffer is used to provide a way to access binary asset files through the
// runtime memory.
type coffer struct {
	assets  map[string]*Asset
	watcher *fsnotify.Watcher
}

// cofferSingleton is the singleton instance of the `coffer`.
var cofferSingleton = &coffer{
	assets: map[string]*Asset{},
}

// init initializes the c.
func (c *coffer) init() error {
	if _, err := os.Stat(AssetRoot); os.IsNotExist(err) {
		return nil
	}

	ar, err := filepath.Abs(AssetRoot)
	if err != nil {
		return err
	}

	dirs, files, err := walkFiles(ar, AssetExts)
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

		if mt := mime.TypeByExtension(filepath.Ext(file)); mt != "" {
			b, err = minifierSingleton.minify(mt, b)
			if err != nil {
				return err
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

// watchTemplates watchs the changing of all asset files.
func (c *coffer) watchAssets() {
	for {
		select {
		case event := <-c.watcher.Events:
			if CofferEnabled {
				INFO(event)
			}

			if event.Op == fsnotify.Create {
				c.watcher.Add(event.Name)
			}

			if err := c.init(); err != nil && CofferEnabled {
				ERROR(err)
			}
		case err := <-c.watcher.Errors:
			if CofferEnabled {
				ERROR(err)
			}
		}
	}
}

// asset returns an `Asset` in the `Coffer` for the provided name.
//
// **Please use the `filepath.Abs()` to process the name before using.**
func (c *coffer) asset(name string) *Asset {
	if !CofferEnabled {
		return nil
	}
	return c.assets[name]
}

// Asset is a binary asset file.
type Asset struct {
	Name    string
	ModTime time.Time
	Reader  *bytes.Reader
}
