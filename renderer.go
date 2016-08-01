package air

import (
	"errors"
	"html/template"
	"io"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type Renderer struct {
	goTemplate *template.Template
	funcMap    template.FuncMap
}

var (
	errBadComparisonType = errors.New("invalid type for comparison")
	errBadComparison     = errors.New("incompatible types for comparison")
	errNoComparison      = errors.New("missing argument for comparison")
)

type typeKind int

const (
	invalidKind typeKind = iota
	boolKind
	complexKind
	intKind
	floatKind
	stringKind
	uintKind
)

func (r *Renderer) Render(wr io.Writer, tplName string, c *Context) error {
	return r.goTemplate.ExecuteTemplate(wr, tplName, c.Data)
}

func (r *Renderer) initDefaultTempleFuncs() {
	r.funcMap = make(template.FuncMap)
	r.funcMap["strlen"] = strlen
	r.funcMap["substr"] = substr
	r.funcMap["str2html"] = str2html
	r.funcMap["html2str"] = html2str
	r.funcMap["datefmt"] = datefmt
	r.funcMap["eq"] = eq
	r.funcMap["ne"] = ne
	r.funcMap["lt"] = lt
	r.funcMap["le"] = le
	r.funcMap["gt"] = gt
	r.funcMap["ge"] = ge
}

func (r *Renderer) AddTemplateFunc(name string, f interface{}) {
	r.funcMap[name] = f
	r.goTemplate.Funcs(r.funcMap)
}

func (r *Renderer) parseTemplates(src string) {
	r.initDefaultTempleFuncs()
	if src[len(src)-1] == '/' {
		src = src[:len(src)-1]
	}
	filenames, err := filepath.Glob(src + "/*/*.html")
	if err != nil {
		panic(err)
	}
	for _, filename := range filenames {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		s := string(b)
		// e.g. src == "views", "views/parts/header.html" will be "parts/header.html".
		name := filename[len(src)+1:]
		// First template becomes return value if not already defined,
		// and we use that one for subsequent New calls to associate
		// all the templates together. Also, if this file has the same name
		// as t, this file becomes the contents of t, so
		//  t, err := New(name).Funcs(xxx).ParseFiles(name)
		// works. Otherwise we create a new template associated with t.
		var tmpl *template.Template
		if r.goTemplate == nil {
			r.goTemplate = template.New(name).Funcs(r.funcMap)
		}
		if name == r.goTemplate.Name() {
			tmpl = r.goTemplate
		} else {
			tmpl = r.goTemplate.New(name)
		}
		_, err = tmpl.Parse(s)
		if err != nil {
			panic(err)
		}
	}
}

// strlen returns the number of characters in s.
func strlen(s string) int {
	return len([]rune(s))
}

// substr returns the substring from start to length.
func substr(s string, start, length int) string {
	bt := []rune(s)
	if start < 0 {
		start = 0
	}
	if start > len(bt) {
		start = start % len(bt)
	}
	var end int
	if (start + length) > (len(bt) - 1) {
		end = len(bt)
	} else {
		end = start + length
	}
	return string(bt[start:end])
}

// html2str returns escaping text convert from html.
func html2str(html string) string {
	src := string(html)

	re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllStringFunc(src, strings.ToLower)

	//remove STYLE
	re, _ = regexp.Compile("\\<style[\\S\\s]+?\\</style\\>")
	src = re.ReplaceAllString(src, "")

	//remove SCRIPT
	re, _ = regexp.Compile("\\<script[\\S\\s]+?\\</script\\>")
	src = re.ReplaceAllString(src, "")

	re, _ = regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllString(src, "\n")

	re, _ = regexp.Compile("\\s{2,}")
	src = re.ReplaceAllString(src, "\n")

	return strings.TrimSpace(src)
}

// str2html returns the `template.HTML` convert from raw.
func str2html(raw string) template.HTML {
	return template.HTML(raw)
}

// datefmt takes a time and a layout string and returns a string with the formatted date.
func datefmt(t time.Time, layout string) string {
	return t.Format(layout)
}

// eq evaluates the comparison a == b || a == c || ...
func eq(arg1 interface{}, arg2 ...interface{}) (bool, error) {
	v1 := reflect.ValueOf(arg1)
	k1, err := basicKind(v1)
	if err != nil {
		return false, err
	}
	if len(arg2) == 0 {
		return false, errNoComparison
	}
	for _, arg := range arg2 {
		v2 := reflect.ValueOf(arg)
		k2, err := basicKind(v2)
		if err != nil {
			return false, err
		}
		if k1 != k2 {
			return false, errBadComparison
		}
		truth := false
		switch k1 {
		case boolKind:
			truth = v1.Bool() == v2.Bool()
		case complexKind:
			truth = v1.Complex() == v2.Complex()
		case floatKind:
			truth = v1.Float() == v2.Float()
		case intKind:
			truth = v1.Int() == v2.Int()
		case stringKind:
			truth = v1.String() == v2.String()
		case uintKind:
			truth = v1.Uint() == v2.Uint()
		default:
			panic("invalid kind")
		}
		if truth {
			return true, nil
		}
	}
	return false, nil
}

// ne evaluates the comparison a != b.
func ne(arg1, arg2 interface{}) (bool, error) {
	// != is the inverse of ==.
	equal, err := eq(arg1, arg2)
	return !equal, err
}

// lt evaluates the comparison a < b.
func lt(arg1, arg2 interface{}) (bool, error) {
	v1 := reflect.ValueOf(arg1)
	k1, err := basicKind(v1)
	if err != nil {
		return false, err
	}
	v2 := reflect.ValueOf(arg2)
	k2, err := basicKind(v2)
	if err != nil {
		return false, err
	}
	if k1 != k2 {
		return false, errBadComparison
	}
	truth := false
	switch k1 {
	case boolKind, complexKind:
		return false, errBadComparisonType
	case floatKind:
		truth = v1.Float() < v2.Float()
	case intKind:
		truth = v1.Int() < v2.Int()
	case stringKind:
		truth = v1.String() < v2.String()
	case uintKind:
		truth = v1.Uint() < v2.Uint()
	default:
		panic("invalid kind")
	}
	return truth, nil
}

// le evaluates the comparison <= b.
func le(arg1, arg2 interface{}) (bool, error) {
	// <= is < or ==.
	lessThan, err := lt(arg1, arg2)
	if lessThan || err != nil {
		return lessThan, err
	}
	return eq(arg1, arg2)
}

// gt evaluates the comparison a > b.
func gt(arg1, arg2 interface{}) (bool, error) {
	// > is the inverse of <=.
	lessOrEqual, err := le(arg1, arg2)
	if err != nil {
		return false, err
	}
	return !lessOrEqual, nil
}

// ge evaluates the comparison a >= b.
func ge(arg1, arg2 interface{}) (bool, error) {
	// >= is the inverse of <.
	lessThan, err := lt(arg1, arg2)
	if err != nil {
		return false, err
	}
	return !lessThan, nil
}

func basicKind(v reflect.Value) (typeKind, error) {
	switch v.Kind() {
	case reflect.Bool:
		return boolKind, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intKind, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintKind, nil
	case reflect.Float32, reflect.Float64:
		return floatKind, nil
	case reflect.Complex64, reflect.Complex128:
		return complexKind, nil
	case reflect.String:
		return stringKind, nil
	}
	return invalidKind, errBadComparisonType
}
