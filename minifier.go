package air

import (
	"bytes"
	"errors"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/json"
	"github.com/tdewolff/minify/svg"
	"github.com/tdewolff/minify/xml"
)

// minifier is used to minify contents by the MIME types.
type minifier struct {
	minifier *minify.M
}

// minifierSingleton is the singleton instance of the `minifier`.
var minifierSingleton = &minifier{
	minifier: minify.New(),
}

// minify minifies the b by the mimeType.
func (m *minifier) minify(mimeType string, b []byte) ([]byte, error) {
	if ss := strings.Split(mimeType, ";"); len(ss) > 1 {
		mimeType = ss[0]
	}
	buf := &bytes.Buffer{}
	if err := m.minifier.Minify(
		mimeType,
		buf,
		bytes.NewReader(b),
	); err == minify.ErrNotExist {
		switch mimeType {
		case "text/html":
			m.minifier.Add(mimeType, html.DefaultMinifier)
		case "text/css":
			m.minifier.Add(mimeType, css.DefaultMinifier)
		case "text/javascript":
			m.minifier.Add(mimeType, js.DefaultMinifier)
		case "application/json":
			m.minifier.Add(mimeType, json.DefaultMinifier)
		case "text/xml":
			m.minifier.Add(mimeType, xml.DefaultMinifier)
		case "image/svg+xml":
			m.minifier.Add(mimeType, svg.DefaultMinifier)
		case "image/jpeg":
			m.minifier.AddFunc(mimeType, func(
				m *minify.M,
				w io.Writer,
				r io.Reader,
				params map[string]string,
			) error {
				img, err := jpeg.Decode(r)
				if err != nil {
					return err
				}
				return jpeg.Encode(w, img, nil)
			})
		case "image/png":
			m.minifier.AddFunc(mimeType, func(
				m *minify.M,
				w io.Writer,
				r io.Reader,
				params map[string]string,
			) error {
				img, err := png.Decode(r)
				if err != nil {
					return err
				}
				return (&png.Encoder{
					CompressionLevel: png.BestCompression,
				}).Encode(w, img)
			})
		default:
			return nil, errors.New("unsupported mime type")
		}
		return m.minify(mimeType, b)
	} else if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
