package air

import (
	"bytes"
	"errors"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/json"
	"github.com/tdewolff/minify/svg"
	"github.com/tdewolff/minify/xml"
)

// minifier is used to provide a `Minify()` method for an `Air` instance
// for minifies a content by a MIME type.
type minifier struct {
	pngEncoder *png.Encoder
	m          *minify.M
}

// newMinifier returns a pointer of a new instance of the `minifier`.
func newMinifier() *minifier {
	return &minifier{
		pngEncoder: &png.Encoder{
			CompressionLevel: png.BestCompression,
		},
		m: minify.New(),
	}
}

// init initializes the `Minifier`. It will be called in the `Air#Serve()`.
func (m *minifier) init() error {
	m.m.Add("text/html", &html.Minifier{})

	m.m.Add("text/css", &css.Minifier{
		Decimals: -1,
	})

	m.m.Add("text/javascript", &js.Minifier{})

	m.m.Add("application/json", &json.Minifier{})

	m.m.Add("text/xml", &xml.Minifier{})

	m.m.Add("image/svg+xml", &svg.Minifier{
		Decimals: -1,
	})

	return nil
}

// minify minifies the b by the mimeType.
func (m *minifier) minify(mimeType string, b []byte) ([]byte, error) {
	if ss := strings.Split(mimeType, ";"); len(ss) > 1 {
		mimeType = ss[0]
	}
	switch mimeType {
	case "image/jpeg":
		return m.minifyJPEG(b)
	case "image/png":
		return m.minifyPNG(b)
	}
	return m.minifyOthers(mimeType, b)
}

// minifyJPEG minifies the b by using the "image/jpeg".
func (m *minifier) minifyJPEG(b []byte) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// minifyPNG minifies the b by using the "image/png".
func (m *minifier) minifyPNG(b []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := m.pngEncoder.Encode(buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// minifyOthers minifies the b by the mimeType by using the
// "github.com/tdewolff/minify".
func (m *minifier) minifyOthers(mimeType string, b []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := m.m.Minify(mimeType, buf, bytes.NewReader(b))
	if err == minify.ErrNotExist {
		return nil, errors.New("unsupported mime type")
	} else if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
