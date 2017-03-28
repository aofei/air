package air

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCofferInit(t *testing.T) {
	html := `
<!DOCTYPE html>
<html>
	<head>
		<title>Air Web Framework</title>
	</head>

	<body>
		<h1>Hello, I am the Air.</h1>
	</body>
</html>
`
	css := `
body {
	font-size: 16px;
}
`
	js := `
alert("Hello, I am the Air.");
`
	json := `
{
	"name": "Air",
	"author": "Aofei Sheng",
}
`
	xml := `
<Info>
	<Name>Air</Name>
	<Author>Aofei Sheng</Author>
</Info>
`
	svg := `
<svg width="100%" height="100%" version="1.1" xmlns="http://www.w3.org/2000/svg">
	<rect width="10" height="10" style="fill:rgb(0,0,255);stroke-width:1; stroke:rgb(0,0,0)"/>
</svg>
`
	txt := `
Hello, I am the Air.
`

	assets := "assets"

	os.Mkdir(assets, os.ModePerm)
	defer func() {
		os.Remove(assets)
	}()

	htmlFile, _ := os.Create(assets + "/index.html")
	defer func() {
		htmlFile.Close()
		os.Remove(htmlFile.Name())
	}()
	htmlFile.WriteString(html)

	cssFile, _ := os.Create(assets + "/main.css")
	defer func() {
		cssFile.Close()
		os.Remove(cssFile.Name())
	}()
	cssFile.WriteString(css)

	jsFile, _ := os.Create(assets + "/main.js")
	defer func() {
		jsFile.Close()
		os.Remove(jsFile.Name())
	}()
	jsFile.WriteString(js)

	jsonFile, _ := os.Create(assets + "/info.json")
	defer func() {
		jsonFile.Close()
		os.Remove(jsonFile.Name())
	}()
	jsonFile.WriteString(json)

	xmlFile, _ := os.Create(assets + "/info.xml")
	defer func() {
		xmlFile.Close()
		os.Remove(xmlFile.Name())
	}()
	xmlFile.WriteString(xml)

	svgFile, _ := os.Create(assets + "/rect.svg")
	defer func() {
		svgFile.Close()
		os.Remove(svgFile.Name())
	}()
	svgFile.WriteString(svg)

	txtFile, _ := os.Create(assets + "/info.txt")
	defer func() {
		txtFile.Close()
		os.Remove(txtFile.Name())
	}()
	txtFile.WriteString(txt)

	a := New()
	a.Minifier.Init()

	a.Config.CofferEnabled = true
	a.Config.AssetExts = []string{".html", ".css", ".js", ".json", ".xml", ".svg", ".txt"}
	a.Config.AssetMinified = true

	assert.NoError(t, a.Coffer.Init())

	abs, _ := filepath.Abs(htmlFile.Name())
	assert.NotNil(t, a.Coffer.Asset(abs))
	assert.True(t, a.Coffer.Asset(abs).reader.Len() < len(html))

	abs, _ = filepath.Abs(cssFile.Name())
	assert.NotNil(t, a.Coffer.Asset(abs))
	assert.True(t, a.Coffer.Asset(abs).reader.Len() < len(css))

	abs, _ = filepath.Abs(jsFile.Name())
	assert.NotNil(t, a.Coffer.Asset(abs))
	assert.True(t, a.Coffer.Asset(abs).reader.Len() < len(js))

	abs, _ = filepath.Abs(jsonFile.Name())
	assert.NotNil(t, a.Coffer.Asset(abs))
	assert.True(t, a.Coffer.Asset(abs).reader.Len() < len(json))

	abs, _ = filepath.Abs(xmlFile.Name())
	assert.NotNil(t, a.Coffer.Asset(abs))
	assert.True(t, a.Coffer.Asset(abs).reader.Len() < len(xml))

	abs, _ = filepath.Abs(svgFile.Name())
	assert.NotNil(t, a.Coffer.Asset(abs))
	assert.True(t, a.Coffer.Asset(abs).reader.Len() < len(svg))

	abs, _ = filepath.Abs(txtFile.Name())
	assert.NotNil(t, a.Coffer.Asset(abs))
	assert.True(t, a.Coffer.Asset(abs).reader.Len() == len(txt))
}
