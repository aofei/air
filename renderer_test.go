package air

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRendererSetTemplateFunc(t *testing.T) {
	a := New()
	r := a.Renderer.(*renderer)
	r.SetTemplateFunc("unixnano", func() int64 { return time.Now().UnixNano() })
	assert.NotNil(t, r.templateFuncMap["unixnano"])
}

func TestRendererInitAndRender(t *testing.T) {
	index := `
<!DOCTYPE html>
<html>
<head>
<title>The Air Web Framework</title>
</head>

<body>
{{template "parts/header.html" .}}
{{template "parts/main.html" .}}
{{template "parts/footer.html" .}}
</body>
</html>
`
	header := `
<header>
<p>Here is the header.</p>
</header>
`
	main := `
<main>
<p>Here is the main.</p>
</main>
`
	footer := `
<footer>
<p>Here is the footer.</p>
</footer>
`
	result := `<!doctype html><title>The Air Web Framework</title>
<header>
<p>Here is the header.
</header>
<main>
<p>Here is the main.
</main>
<footer>
<p>Here is the footer.
</footer>
`

	templates := "templates"
	templatesParts := templates + "/parts"

	os.Mkdir(templates, os.ModePerm)
	defer func() {
		os.Remove(templates)
	}()

	os.Mkdir(templatesParts, os.ModePerm)
	defer func() {
		os.Remove(templatesParts)
	}()

	indexFile, _ := os.Create(templates + "/index.html")
	defer func() {
		indexFile.Close()
		os.Remove(indexFile.Name())
	}()
	indexFile.WriteString(index)

	headerFile, _ := os.Create(templatesParts + "/header.html")
	defer func() {
		indexFile.Close()
		os.Remove(headerFile.Name())
	}()
	headerFile.WriteString(header)

	mainFile, _ := os.Create(templatesParts + "/main.html")
	defer func() {
		mainFile.Close()
		os.Remove(mainFile.Name())
	}()
	mainFile.WriteString(main)

	a := New()
	a.Minifier.Init()

	a.Config.TemplateMinified = true

	b := &bytes.Buffer{}

	assert.NoError(t, a.Renderer.Init())
	assert.Error(t, a.Renderer.Render(b, "index.html", nil))

	footerFile, _ := os.Create(templatesParts + "/footer.html")
	defer func() {
		footerFile.Close()
		os.Remove(footerFile.Name())
	}()
	footerFile.WriteString(footer)

	time.Sleep(time.Millisecond) // Wait for renderer
	assert.NoError(t, a.Renderer.Render(b, "index.html", nil))
	assert.Equal(t, result, b.String())
}

func TestRendererTemplateFuncs(t *testing.T) {
	assert.Equal(t, 9, strlen("Hello, 世界"))
	assert.Equal(t, "The Air Web Framework", strcat("The ", "Air ", "Web ", "Framework"))
	assert.Equal(t, "世界", substr("Hello, 世界", 7, 9))

	str := "2016-07-20T12:13:54Z"
	tm, _ := time.Parse(time.RFC3339, str)
	assert.Equal(t, str, timefmt(tm, time.RFC3339))
}
