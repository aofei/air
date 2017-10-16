package air

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type (
	// Binder is used to provide a `Bind()` method for an `Air` instance for
	// binds an HTTP request body into privided type.
	Binder interface {
		// Bind binds the body of the req into the provided type i.
		Bind(i interface{}, req *Request) error
	}

	// binder implements the `Binder` based on the "Content-Type" header.
	binder struct{}
)

// newBinder returns a pointer of a new instance of the `binder`.
func newBinder() *binder {
	return &binder{}
}

// Bind implements the `Binder#Bind()` based on the "Content-Type" header.
func (b *binder) Bind(i interface{}, req *Request) error {
	if req.Method == "GET" {
		err := b.bindData(i, req.URL.Query(), "query")
		if err != nil {
			err = NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return err
	} else if req.Body == nil {
		return NewHTTPError(
			http.StatusBadRequest,
			"request body can't be empty",
		)
	}

	ctype := req.Header.Get("Content-Type")

	var err error
	err = ErrUnsupportedMediaType

	switch {
	case strings.HasPrefix(ctype, "application/json"):
		if err = json.NewDecoder(req.Body).Decode(i); err != nil {
			if ute, ok := err.(*json.UnmarshalTypeError); ok {
				err = NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"unmarshal type error: "+
							"expected=%v, got=%v, "+
							"offset=%v",
						ute.Type,
						ute.Value,
						ute.Offset,
					),
				)
			} else if se, ok := err.(*json.SyntaxError); ok {
				err = NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"syntax error: offset=%v, "+
							"error=%v",
						se.Offset,
						se.Error(),
					),
				)
			} else {
				err = NewHTTPError(
					http.StatusBadRequest,
					err.Error(),
				)
			}
		}
	case strings.HasPrefix(ctype, "application/xml"):
		if err = xml.NewDecoder(req.Body).Decode(i); err != nil {
			if ute, ok := err.(*xml.UnsupportedTypeError); ok {
				err = NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"unsupported type error: "+
							"type=%v, error=%v",
						ute.Type,
						ute.Error(),
					),
				)
			} else if se, ok := err.(*xml.SyntaxError); ok {
				err = NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"syntax error: line=%v, "+
							"error=%v",
						se.Line,
						se.Error(),
					),
				)
			} else {
				err = NewHTTPError(
					http.StatusBadRequest,
					err.Error(),
				)
			}
		}
	case strings.HasPrefix(ctype, "application/x-www-form-urlencoded"),
		strings.HasPrefix(ctype, "multipart/form-data"):
		if err = req.ParseForm(); err == nil {
			if err = b.bindData(i, req.Form, "form"); err != nil {
				err = NewHTTPError(
					http.StatusBadRequest,
					err.Error(),
				)
			}
		}
	}

	return err
}

// bindData binds the data into the type ptr with the tag.
func (b *binder) bindData(ptr interface{}, data url.Values, tag string) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	if typ.Kind() != reflect.Struct {
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)

		if !structField.CanSet() {
			continue
		}

		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)

		if inputFieldName == "" {
			inputFieldName = typeField.Name
			// If tag is nil, we inspect if the field is a struct.
			if structFieldKind == reflect.Struct {
				if err := b.bindData(
					structField.Addr().Interface(),
					data,
					tag,
				); err != nil {
					return err
				}
				continue
			}
		}

		inputValue, exists := data[inputFieldName]

		if !exists {
			continue
		}

		numElems := len(inputValue)

		if structFieldKind == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(
				structField.Type(),
				numElems,
				numElems,
			)

			for i := 0; i < numElems; i++ {
				if err := setWithProperType(
					sliceOf,
					inputValue[i],
					slice.Index(i),
				); err != nil {
					return err
				}
			}

			val.Field(i).Set(slice)
		} else {
			if err := setWithProperType(
				typeField.Type.Kind(),
				inputValue[0],
				structField,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

// setWithProperType sets the val into a field with a proper k.
func setWithProperType(k reflect.Kind, val string, field reflect.Value) error {
	bitSize := 0
	switch k {
	case reflect.Int8, reflect.Uint8:
		bitSize = 8
	case reflect.Int16, reflect.Uint16:
		bitSize = 16
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		bitSize = 32
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		bitSize = 64
	}

	switch k {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		return setIntField(val, bitSize, field)
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return setUintField(val, bitSize, field)
	case reflect.Bool:
		return setBoolField(val, field)
	case reflect.Float32, reflect.Float64:
		return setFloatField(val, bitSize, field)
	case reflect.String:
		field.SetString(val)
	default:
		return errors.New("unknown type")
	}
	return nil
}

// setIntField sets the value into a field with a provided bitSize.
func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

// setUintField sets the value into a field with a provided bitSize.
func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

// setBoolField sets the value into a field.
func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

// setFloatField sets the value into a field with a provided bitSize.
func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}
