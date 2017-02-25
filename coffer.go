package air

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/json"
	"github.com/tdewolff/minify/svg"
	"github.com/tdewolff/minify/xml"
)

type (
	// Coffer is used to provide an `Asset()` method for an `Air` instance for access binary
	// asset files by using the runtime memory.
	Coffer interface {
		// LoadAssets loads asset files. It will be called in the `Air#Serve()`.
		LoadAssets() error

		// Asset returns an `Asset` in the `Coffer` for the provided name.
		Asset(name string) *Asset
	}

	// Asset is a binary asset file.
	Asset struct {
		name    string
		modTime time.Time
		reader  *bytes.Reader
	}

	// coffer implements the `Coffer` by using the `map[string]*Asset`.
	coffer struct {
		air *Air

		assets   map[string]*Asset
		minifier *minify.M
		watcher  *fsnotify.Watcher
	}
)

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

// newAsset returns a pointer of a new instance of the `coffer`.
func newCoffer(a *Air) *coffer {
	return &coffer{
		air:    a,
		assets: make(map[string]*Asset),
	}
}

// LoadAssets implements the `Coffer#LoadAssets()` by using the `map[string]*Asset`.
func (c *coffer) LoadAssets() error {
	cfg := c.air.Config

	if !cfg.CofferEnabled {
		return nil
	} else if _, err := os.Stat(cfg.AssetRoot); err != nil && os.IsNotExist(err) {
		return nil
	}

	ar, err := filepath.Abs(cfg.AssetRoot)
	if err != nil {
		return err
	}

	dirs, err := walkDirs(ar)
	if err != nil {
		return err
	}

	var filenames []string
	for _, dir := range dirs {
		for _, ae := range cfg.AssetExts {
			fns, err := filepath.Glob(filepath.Join(dir, "*"+ae))
			if err != nil {
				return err
			}
			filenames = append(filenames, fns...)
		}
	}

	assets := make(map[string]*Asset)

	for _, filename := range filenames {
		fi, err := os.Stat(filename)
		if err != nil {
			return err
		}

		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}

		if cfg.AssetMinified {
			if c.minifier == nil {
				c.minifier = minify.New()

				c.minifier.Add("text/html", &html.Minifier{
					KeepDefaultAttrVals: true,
					KeepDocumentTags:    true,
					KeepWhitespace:      true,
				})

				c.minifier.Add("text/css", &css.Minifier{
					Decimals: -1,
				})

				c.minifier.Add("text/javascript", &js.Minifier{})

				c.minifier.Add("application/json", &json.Minifier{})

				c.minifier.Add("text/xml", &xml.Minifier{
					KeepWhitespace: true,
				})

				c.minifier.Add("image/svg+xml", &svg.Minifier{
					Decimals: -1,
				})
			}

			buf := &bytes.Buffer{}

			switch ext := filepath.Ext(filename); ext {
			case ".html":
				err = c.minifier.Minify("text/html", buf, bytes.NewReader(b))
			case ".css":
				err = c.minifier.Minify("text/css", buf, bytes.NewReader(b))
			case ".js":
				err = c.minifier.Minify("text/javascript", buf, bytes.NewReader(b))
			case ".json":
				err = c.minifier.Minify("application/json", buf, bytes.NewReader(b))
			case ".xml":
				err = c.minifier.Minify("text/xml", buf, bytes.NewReader(b))
			case ".svg":
				err = c.minifier.Minify("image/svg+xml", buf, bytes.NewReader(b))
			}

			if err != nil {
				return err
			}

			b = buf.Bytes()
		}

		assets[filename] = NewAsset(filename, fi.ModTime(), b)
	}

	c.assets = assets

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

	return nil
}

// Asset implements the `Coffer#Asset()` by using the `map[string]*Asset`.
func (c *coffer) Asset(name string) *Asset {
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

			if err := c.LoadAssets(); err != nil {
				c.air.Logger.Error(err)
			}
		case err := <-c.watcher.Errors:
			c.air.Logger.Error(err)
		}
	}
}
