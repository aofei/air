package gases

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/sheng/air"
)

type (
	// StaticConfig defines the config for Static gas.
	StaticConfig struct {
		// Skipper defines a function to skip gas.
		Skipper Skipper

		// Root directory from where the static content is served.
		// Required.
		Root string `json:"root"`

		// Index file for serving a directory.
		// Optional. Default value "index.html".
		Index string `json:"index"`

		// Enable HTML5 mode by forwarding all not-found requests to root so that
		// SPA (single-page application) can handle the routing.
		// Optional. Default value false.
		HTML5 bool `json:"html5"`

		// Enable directory browsing.
		// Optional. Default value false.
		Browse bool `json:"browse"`
	}
)

var (
	// DefaultStaticConfig is the default Static gas config.
	DefaultStaticConfig = StaticConfig{
		Skipper: defaultSkipper,
		Index:   "index.html",
	}
)

// Static returns a Static gas to serves static content from the provided
// root directory.
func Static(root string) air.GasFunc {
	c := DefaultStaticConfig
	c.Root = root
	return StaticWithConfig(c)
}

// StaticWithConfig returns a Static gas from config.
// See `Static()`.
func StaticWithConfig(config StaticConfig) air.GasFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultStaticConfig.Skipper
	}
	if config.Index == "" {
		config.Index = DefaultStaticConfig.Index
	}

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c *air.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			fs := http.Dir(config.Root)
			p := c.Request.URI.Path()
			if strings.Contains(c.Path, "*") { // If serving from a group, e.g. `/static*`.
				p = c.Params[c.ParamNames[0]]
			}
			file := path.Clean(p)
			f, err := fs.Open(file)
			if err != nil {
				// HTML5 mode
				err = next(c)
				if he, ok := err.(*air.HTTPError); ok {
					if config.HTML5 && he.Code == air.ErrNotFound.Code {
						file = ""
						f, err = fs.Open(file)
					} else {
						return err
					}
				} else {
					return err
				}
			}
			defer f.Close()
			fi, err := f.Stat()
			if err != nil {
				return err
			}

			if fi.IsDir() {
				/* NOTE:
				Not checking the Last-Modified header as it caches the response `304` when
				changing different directories for the same path.
				*/
				d := f

				// Index file
				file = path.Join(file, config.Index)
				f, err = fs.Open(file)
				if err == nil {
					// Index file
					if fi, err = f.Stat(); err != nil {
						return err
					}
				} else if err != nil && config.Browse {
					dirs, err := d.Readdir(-1)
					if err != nil {
						return err
					}

					// Create a directory index
					res := c.Response
					res.Header.Set(air.HeaderContentType, air.MIMETextHTML)
					if _, err = fmt.Fprintf(res, "<pre>\n"); err != nil {
						return err
					}
					for _, d := range dirs {
						name := d.Name()
						color := "#212121"
						if d.IsDir() {
							color = "#e91e63"
							name += "/"
						}
						if _, err = fmt.Fprintf(res, "<a href=\"%s\" style=\"color: %s;\">%s</a>\n", name, color, name); err != nil {
							return err
						}
					}
					_, err = fmt.Fprintf(res, "</pre>\n")
					return err
				} else {
					return next(c)
				}
			}
			return c.ServeContent(f, fi.Name(), fi.ModTime())
		}
	}
}
