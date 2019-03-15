package air

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/aofei/mimesniffer"
	"github.com/cespare/xxhash/v2"
	"github.com/fsnotify/fsnotify"
)

// coffer is a binary asset file manager that uses runtime memory to reduce disk
// I/O pressure.
type coffer struct {
	a         *Air
	loadOnce  *sync.Once
	loadError error
	watcher   *fsnotify.Watcher
	assets    *sync.Map
	cache     *fastcache.Cache
}

// newCoffer returns a new instance of the `coffer` with the a.
func newCoffer(a *Air) *coffer {
	return &coffer{
		a:        a,
		loadOnce: &sync.Once{},
	}
}

// load loads the stuff of the c up.
func (c *coffer) load() {
	defer func() {
		if c.loadError != nil {
			c.loadOnce = &sync.Once{}
		}
	}()

	if c.watcher == nil {
		c.watcher, c.loadError = fsnotify.NewWatcher()
		if c.loadError != nil {
			return
		}

		go func() {
			for {
				select {
				case e := <-c.watcher.Events:
					ai, ok := c.assets.Load(e.Name)
					if !ok {
						break
					}

					a := ai.(*asset)
					c.assets.Delete(a.name)
					c.cache.Del(a.digest)
					if a.gzippedDigest != nil {
						c.cache.Del(a.gzippedDigest)
					}
				case err := <-c.watcher.Errors:
					c.a.errorLogger.Printf(
						"air: coffer watcher error: %v",
						err,
					)
				}
			}
		}()
	}

	c.assets = &sync.Map{}
	c.cache = fastcache.New(c.a.CofferMaxMemoryBytes)
}

// asset returns an `asset` from the c for the name.
func (c *coffer) asset(name string) (*asset, error) {
	if c.loadOnce.Do(c.load); c.loadError != nil {
		return nil, c.loadError
	} else if ai, ok := c.assets.Load(name); ok {
		return ai.(*asset), nil
	} else if ar, err := filepath.Abs(c.a.CofferAssetRoot); err != nil {
		return nil, err
	} else if !strings.HasPrefix(name, ar) {
		return nil, nil
	}

	ext := filepath.Ext(name)
	if !stringSliceContainsCIly(c.a.CofferAssetExts, ext) {
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

	if mt == "" {
		mt = mimesniffer.Sniff(b)
	}

	pmt, _, err := mime.ParseMediaType(mt)
	if err != nil {
		return nil, err
	}

	if c.a.MinifierEnabled &&
		stringSliceContainsCIly(c.a.MinifierMIMETypes, pmt) {
		if b, err = c.a.minifier.minify(pmt, b); err != nil {
			return nil, err
		}

		minified = true
	}

	if c.a.GzipEnabled && int64(len(b)) >= c.a.GzipMinContentLength &&
		stringSliceContainsCIly(c.a.GzipMIMETypes, pmt) {
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
	c.cache.SetBig(a.digest, b)

	if gb != nil {
		a.gzippedDigest = make([]byte, 8)
		binary.BigEndian.PutUint64(a.gzippedDigest, xxhash.Sum64(gb))
		c.cache.SetBig(a.gzippedDigest, gb)
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
		c = a.coffer.cache.GetBig(nil, a.gzippedDigest)
	} else {
		c = a.coffer.cache.GetBig(nil, a.digest)
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
