package air

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"mime"
	"reflect"
	"strconv"
)

// binder is a binder that binds request based on the MIME types.
type binder struct{}

// theBinder is the singleton of the `binder`.
var theBinder = &binder{}

// bind binds the r into the v.
func (b *binder) bind(v interface{}, r *Request) error {
	if r.Method == "GET" {
		return b.bindParams(v, r.Params)
	} else if r.Body == nil {
		return &Error{
			Code:    400,
			Message: "request body can't be empty",
		}
	}

	mt, _, err := mime.ParseMediaType(r.Headers["Content-Type"])
	if err != nil {
		return &Error{
			Code:    400,
			Message: err.Error(),
		}
	}

	switch mt {
	case "application/json":
		err = json.NewDecoder(r.Body).Decode(v)
	case "application/xml":
		err = xml.NewDecoder(r.Body).Decode(v)
	case "application/x-www-form-urlencoded", "multipart/form-data":
		err = b.bindParams(v, r.Params)
	default:
		return &Error{
			Code:    415,
			Message: "Unsupported Media Type",
		}
	}

	if err != nil {
		return &Error{
			Code:    400,
			Message: err.Error(),
		}
	}

	return nil
}

// bindParams binds the params into the v.
func (b *binder) bindParams(v interface{}, params map[string]string) error {
	typ := reflect.TypeOf(v).Elem()
	if typ.Kind() != reflect.Struct {
		return errors.New("binding element must be a struct")
	}

	val := reflect.ValueOf(v).Elem()
	for i := 0; i < typ.NumField(); i++ {
		vf := val.Field(i)
		if !vf.CanSet() {
			continue
		}

		vfk := vf.Kind()
		if vfk == reflect.Struct {
			err := b.bindParams(vf.Addr().Interface(), params)
			if err != nil {
				return err
			}

			continue
		}

		tf := typ.Field(i)

		p, ok := params[tf.Name]
		if !ok {
			continue
		}

		switch tf.Type.Kind() {
		case reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64:
			if p == "" {
				p = "0"
			}

			v, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				return err
			}

			vf.SetInt(v)
		case reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64:
			if p == "" {
				p = "0"
			}

			v, err := strconv.ParseUint(p, 10, 64)
			if err != nil {
				return err
			}

			vf.SetUint(v)
		case reflect.Bool:
			if p == "" {
				p = "false"
			}

			v, err := strconv.ParseBool(p)
			if err != nil {
				return err
			}

			vf.SetBool(v)
		case reflect.Float32, reflect.Float64:
			if p == "" {
				p = "0.0"
			}

			v, err := strconv.ParseFloat(p, 64)
			if err != nil {
				return err
			}

			vf.SetFloat(v)
		case reflect.String:
			vf.SetString(p)
		default:
			return errors.New("unknown type")
		}
	}

	return nil
}
