package air

import (
	"crypto/sha256"
	"fmt"
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
	a       *Air
	assets  map[string]*asset
	watcher *fsnotify.Watcher
}

// newCoffer returns a new instance of the `coffer` with the a.
func newCoffer(a *Air) *coffer {
	c := &coffer{
		a:      a,
		assets: map[string]*asset{},
	}

	var err error
	if c.watcher, err = fsnotify.NewWatcher(); err != nil {
		panic(fmt.Errorf(
			"air: failed to build coffer watcher: %v",
			err,
		))
	}

	go func() {
		for {
			select {
			case e := <-c.watcher.Events:
				if a.CofferEnabled {
					a.DEBUG(
						"air: asset file event occurs",
						map[string]interface{}{
							"file":  e.Name,
							"event": e.Op.String(),
						},
					)
				}

				delete(c.assets, e.Name)
			case err := <-c.watcher.Errors:
				if a.CofferEnabled {
					a.ERROR(
						"air: coffer watcher error",
						map[string]interface{}{
							"error": err.Error(),
						},
					)
				}
			}
		}
	}()

	return c
}

// asset returns an `asset` from the c for the name.
func (c *coffer) asset(name string) (*asset, error) {
	if !c.a.CofferEnabled {
		return nil, nil
	}

	if a, ok := c.assets[name]; ok {
		return a, nil
	} else if ar, err := filepath.Abs(c.a.AssetRoot); err != nil {
		return nil, err
	} else if !strings.HasPrefix(name, ar) {
		return nil, nil
	}

	ext := filepath.Ext(name)
	if !stringsContainsCIly(c.a.AssetExts, ext) {
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

	mt := mime.TypeByExtension(ext)
	if mt != "" {
		if b, err = c.a.minifier.minify(mt, b); err != nil {
			return nil, err
		}
	}

	if err := c.watcher.Add(name); err != nil {
		return nil, err
	}

	c.assets[name] = &asset{
		name:     name,
		content:  b,
		mimeType: mt,
		checksum: sha256.Sum256(b),
		modTime:  fi.ModTime(),
	}

	return c.assets[name], nil
}

// asset is a binary asset file.
type asset struct {
	name     string
	content  []byte
	mimeType string
	checksum [sha256.Size]byte
	modTime  time.Time
}
