package air

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// binder is a binder that binds request based on the MIME types.
type binder struct{}

// binderSingleton is the singleton of the `binder`.
var binderSingleton = &binder{}

// bind binds the `Body` of the r into the v.
func (b *binder) bind(v interface{}, r *Request) error {
	if r.Method == "GET" {
		err := b.bindValues(v, r.QueryParams, "query")
		if err != nil {
			err = &Error{
				Code:    400,
				Message: err.Error(),
			}
		}
		return err
	} else if r.Body == nil {
		return &Error{
			Code:    400,
			Message: "request body can't be empty",
		}
	}

	ctype := r.Headers["Content-Type"]
	err := error(&Error{
		Code:    415,
		Message: "Unsupported Media Type",
	})

	switch {
	case strings.HasPrefix(ctype, "application/json"):
		if err = json.NewDecoder(r.Body).Decode(v); err != nil {
			if ute, ok := err.(*json.UnmarshalTypeError); ok {
				err = &Error{
					Code: 400,
					Message: fmt.Sprintf(
						"unmarshal type error: "+
							"expected=%v, got=%v, "+
							"offset=%v",
						ute.Type,
						ute.Value,
						ute.Offset,
					),
				}
			} else if se, ok := err.(*json.SyntaxError); ok {
				err = &Error{
					Code: 400,
					Message: fmt.Sprintf(
						"syntax error: offset=%v, "+
							"error=%v",
						se.Offset,
						se.Error(),
					),
				}
			} else {
				err = &Error{
					Code:    400,
					Message: err.Error(),
				}
			}
		}
	case strings.HasPrefix(ctype, "application/xml"):
		if err = xml.NewDecoder(r.Body).Decode(v); err != nil {
			if ute, ok := err.(*xml.UnsupportedTypeError); ok {
				err = &Error{
					Code: 400,
					Message: fmt.Sprintf(
						"unsupported type error: "+
							"type=%v, error=%v",
						ute.Type,
						ute.Error(),
					),
				}
			} else if se, ok := err.(*xml.SyntaxError); ok {
				err = &Error{
					Code: 400,
					Message: fmt.Sprintf(
						"syntax error: line=%v, "+
							"error=%v",
						se.Line,
						se.Error(),
					),
				}
			} else {
				err = &Error{
					Code:    400,
					Message: err.Error(),
				}
			}
		}
	case strings.HasPrefix(ctype, "application/x-www-form-urlencoded"),
		strings.HasPrefix(ctype, "multipart/form-data"):
		if err = b.bindValues(v, r.FormParams, "form"); err != nil {
			err = &Error{
				Code:    400,
				Message: err.Error(),
			}
		}
	}

	return err
}

// bindValues binds the values into the v with the tag.
func (b *binder) bindValues(
	v interface{},
	values map[string]string,
	tag string,
) error {
	typ := reflect.TypeOf(v).Elem()
	val := reflect.ValueOf(v).Elem()

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
				if err := b.bindValues(
					structField.Addr().Interface(),
					values,
					tag,
				); err != nil {
					return err
				}
				continue
			}
		}

		inputValue, exists := values[inputFieldName]

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
					inputValue,
					slice.Index(i),
				); err != nil {
					return err
				}
			}

			val.Field(i).Set(slice)
		} else {
			if err := setWithProperType(
				typeField.Type.Kind(),
				inputValue,
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
