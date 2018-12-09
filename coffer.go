package air

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/cespare/xxhash"
	"github.com/fsnotify/fsnotify"
)

// coffer is a binary asset file manager that uses runtime memory to reduce disk
// I/O pressure.
type coffer struct {
	a       *Air
	once    *sync.Once
	assets  *sync.Map
	cache   *fastcache.Cache
	watcher *fsnotify.Watcher
}

// newCoffer returns a new instance of the `coffer` with the a.
func newCoffer(a *Air) *coffer {
	c := &coffer{
		a:      a,
		once:   &sync.Once{},
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

				if ai, ok := c.assets.Load(e.Name); ok {
					a := ai.(*asset)
					c.assets.Delete(a.name)
					c.cache.Del(a.digest)
					if a.gzippedDigest != nil {
						c.cache.Del(a.gzippedDigest)
					}
				}
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
	c.once.Do(func() {
		c.cache = fastcache.New(c.a.CofferMaxMemoryBytes)
	})

	if ai, ok := c.assets.Load(name); ok {
		return ai.(*asset), nil
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

	var (
		mt       = mime.TypeByExtension(ext)
		minified bool
		gb       []byte
	)

	if mt != "" {
		mt, _, err := mime.ParseMediaType(mt)
		if err != nil {
			return nil, err
		}

		if c.a.MinifierEnabled &&
			stringSliceContains(c.a.MinifierMIMETypes, mt) {
			if b, err = c.a.minifier.minify(mt, b); err != nil {
				return nil, err
			}

			minified = true
		}

		if c.a.GzipEnabled &&
			stringSliceContains(c.a.GzipMIMETypes, mt) {
			buf := bytes.Buffer{}
			if gw, err := gzip.NewWriterLevel(
				&buf,
				c.a.GzipCompressionLevel,
			); err != nil {
				return nil, err
			} else if _, err = gw.Write(b); err != nil {
				return nil, err
			} else if err = gw.Close(); err != nil {
				return nil, err
			}

			gb = buf.Bytes()
		}
	}

	if err := c.watcher.Add(name); err != nil {
		return nil, err
	}

	a := &asset{
		coffer:   c,
		name:     name,
		mimeType: mt,
		modTime:  fi.ModTime(),
		minified: minified,
		digest:   make([]byte, 8),
	}

	binary.BigEndian.PutUint64(a.digest, xxhash.Sum64(b))
	c.cache.Set(a.digest, b)

	if gb != nil {
		a.gzippedDigest = make([]byte, 8)
		binary.BigEndian.PutUint64(a.gzippedDigest, xxhash.Sum64(gb))
		c.cache.Set(a.gzippedDigest, gb)
	}

	c.assets.Store(name, a)

	return a, nil
}

// asset is a binary asset file.
type asset struct {
	coffer        *coffer
	name          string
	mimeType      string
	modTime       time.Time
	minified      bool
	digest        []byte
	gzippedDigest []byte
}

// content returns the content of the a with the gzipped.
func (a *asset) content(gzipped bool) []byte {
	var c []byte
	if gzipped {
		c = a.coffer.cache.Get(nil, a.gzippedDigest)
	} else {
		c = a.coffer.cache.Get(nil, a.digest)
	}

	if len(c) == 0 {
		a.coffer.assets.Delete(a.name)
		a.coffer.cache.Del(a.digest)
		if a.gzippedDigest != nil {
			a.coffer.cache.Del(a.gzippedDigest)
		}

		return nil
	}

	return c
}
