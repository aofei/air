package air

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// coffer is a binary asset file manager that uses runtime memory to reduce disk
// I/O pressure.
type coffer struct {
	a       *Air
	assets  *sync.Map
	watcher *fsnotify.Watcher
}

// newCoffer returns a new instance of the `coffer` with the a.
func newCoffer(a *Air) *coffer {
	c := &coffer{
		a:      a,
		assets: &sync.Map{},
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

				c.assets.Delete(e.Name)
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

	if ai, ok := c.assets.Load(name); ok {
		if a, ok := ai.(*asset); ok {
			return a, nil
		}

		c.assets.Delete(name)
	} else if ar, err := filepath.Abs(c.a.AssetRoot); err != nil {
		return nil, err
	} else if !strings.HasPrefix(name, ar) {
		return nil, nil
	}

	ext := filepath.Ext(name)
	if !stringSliceContainsCIly(c.a.AssetExts, ext) {
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

	a := &asset{
		name:     name,
		content:  b,
		mimeType: mt,
		checksum: sha256.Sum256(b),
		modTime:  fi.ModTime(),
	}

	c.assets.Store(name, a)

	return a, nil
}

// asset is a binary asset file.
type asset struct {
	name     string
	content  []byte
	mimeType string
	checksum [sha256.Size]byte
	modTime  time.Time
}
