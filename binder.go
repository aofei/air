package air

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// Binder is used to provide a `Bind()` method for an `Air` instance
// for binds a HTTP request body into privided type.
type Binder struct {
	air *Air
}

// NewBinder returns a new instance of `Binder`.
func NewBinder(a *Air) *Binder {
	return &Binder{
		air: a,
	}
}

// Bind binds the HTTP request body into provided type i based on
// "Content-Type" header.
func (b *Binder) Bind(i interface{}, c *Context) (err error) {
	req := c.Request
	if req.Method() == GET {
		if err = b.bindData(i, c.QueryParams()); err != nil {
			err = NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return
	}
	ctype := req.Header.Get(HeaderContentType)
	if req.Body() == nil {
		err = NewHTTPError(http.StatusBadRequest, "Request Body Can't Be Empty")
		return
	}
	err = ErrUnsupportedMediaType
	switch {
	case strings.HasPrefix(ctype, MIMEApplicationJSON):
		if err = json.NewDecoder(req.Body()).Decode(i); err != nil {
			err = NewHTTPError(http.StatusBadRequest, err.Error())
		}
	case strings.HasPrefix(ctype, MIMEApplicationXML):
		if err = xml.NewDecoder(req.Body()).Decode(i); err != nil {
			err = NewHTTPError(http.StatusBadRequest, err.Error())
		}
	case strings.HasPrefix(ctype, MIMEApplicationForm), strings.HasPrefix(ctype, MIMEMultipartForm):
		if err = b.bindData(i, req.FormParams()); err != nil {
			err = NewHTTPError(http.StatusBadRequest, err.Error())
		}
	}
	return
}

// bindData binds the data into a type ptr.
func (b *Binder) bindData(ptr interface{}, data map[string][]string) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	if typ.Kind() != reflect.Struct {
		return errors.New("Binding Element Must Be A Struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get("form")

		if inputFieldName == "" {
			inputFieldName = typeField.Name
			// If "form" tag is nil, we inspect if the field is a struct.
			if structFieldKind == reflect.Struct {
				err := b.bindData(structField.Addr().Interface(), data)
				if err != nil {
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
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for i := 0; i < numElems; i++ {
				if err := setWithProperType(sliceOf, inputValue[i], slice.Index(i)); err != nil {
					return err
				}
			}
			val.Field(i).Set(slice)
		} else {
			if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
				return err
			}
		}
	}
	return nil
}

// setWithProperType sets the val into a structField with a proper valueKind.
func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	switch valueKind {
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	default:
		return errors.New("Unknown Type")
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
