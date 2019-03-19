package air

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"mime"
	"net/http"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/golang/protobuf/proto"
	"github.com/vmihailenco/msgpack"
	yaml "gopkg.in/yaml.v2"
)

// binder is a binder that binds request based on the MIME types.
type binder struct {
	a *Air
}

// newBinder returns a new instance of the `binder` with the a.
func newBinder(a *Air) *binder {
	return &binder{
		a: a,
	}
}

// bind binds the r into the v.
func (b *binder) bind(v interface{}, r *Request) error {
	if r.ContentLength == 0 {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodDelete:
			return b.bindParams(v, r.Params())
		}

		r.res.Status = http.StatusBadRequest

		return errors.New("air: request body cannot be empty")
	}

	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return err
	}

	switch mt {
	case "application/json":
		err = json.NewDecoder(r.Body).Decode(v)
	case "application/xml":
		err = xml.NewDecoder(r.Body).Decode(v)
	case "application/protobuf":
		var b []byte
		if b, err = ioutil.ReadAll(r.Body); err == nil {
			err = proto.Unmarshal(b, v.(proto.Message))
		}
	case "application/msgpack":
		err = msgpack.NewDecoder(r.Body).Decode(v)
	case "application/toml":
		_, err = toml.DecodeReader(r.Body, v)
	case "application/yaml":
		err = yaml.NewDecoder(r.Body).Decode(v)
	case "application/x-www-form-urlencoded", "multipart/form-data":
		err = b.bindParams(v, r.Params())
	default:
		r.res.Status = http.StatusUnsupportedMediaType
		err = errors.New(http.StatusText(r.res.Status))
	}

	return err
}

// bindParams binds the ps into the v.
func (b *binder) bindParams(v interface{}, ps []*RequestParam) error {
	t := reflect.TypeOf(v).Elem()
	if t.Kind() != reflect.Struct {
		return errors.New("air: binding element must be a struct")
	}

	val := reflect.ValueOf(v).Elem()
	for i := 0; i < t.NumField(); i++ {
		vf := val.Field(i)
		if !vf.CanSet() {
			continue
		}

		tf := t.Field(i)
		pn := tf.Tag.Get("param")
		if pn == "" {
			if vf.Kind() == reflect.Struct {
				err := b.bindParams(vf.Addr().Interface(), ps)
				if err != nil {
					return err
				}

				continue
			}

			pn = tf.Name
		}

		lpn := strings.ToLower(pn)

		var pv *RequestParamValue
		for _, p := range ps {
			if p.Name == pn {
				pv = p.Value()
				break
			} else if p.Name == lpn && pv == nil {
				pv = p.Value()
			}
		}

		if pv == nil {
			continue
		}

		switch tf.Type.Kind() {
		case reflect.Bool:
			b, err := pv.Bool()
			if err != nil {
				return err
			}

			vf.SetBool(b)
		case reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64:
			i64, err := pv.Int64()
			if err != nil {
				return err
			}

			vf.SetInt(i64)
		case reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64:
			ui64, err := pv.Uint64()
			if err != nil {
				return err
			}

			vf.SetUint(ui64)
		case reflect.Float32, reflect.Float64:
			f64, err := pv.Float64()
			if err != nil {
				return err
			}

			vf.SetFloat(f64)
		case reflect.String:
			vf.SetString(pv.String())
		default:
			return errors.New("air: unknown binding type")
		}
	}

	return nil
}
