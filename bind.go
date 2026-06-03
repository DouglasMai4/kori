package kori

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

const maxBodyBytes = 4 << 20

func BindQuery(r *http.Request, dst any) error {
	if err := decodeQuery(r, dst); err != nil {
		return err
	}

	return validateStruct(dst)
}

func BindPath(r *http.Request, dst any) error {
	if err := decodePath(r, dst); err != nil {
		return err
	}

	return validateStruct(dst)
}

func BindHeader(r *http.Request, dst any) error {
	if err := decodeHeader(r, dst); err != nil {
		return err
	}

	return validateStruct(dst)
}

func BindJSON(r *http.Request, dst any) error {
	if r.Body == nil || r.Body == http.NoBody {
		return validateStruct(dst)
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	defer r.Body.Close()

	if err != nil {
		return BadRequest("cannot read request body", err.Error())
	}

	if len(data) == 0 {
		return validateStruct(dst)
	}

	if err := json.Unmarshal(data, dst); err != nil {
		return BadRequest("invalid JSON body", err.Error())
	}

	return validateStruct(dst)
}

func Bind(r *http.Request, dst any) error {
	if err := decodePath(r, dst); err != nil {
		return err
	}

	if err := decodeQuery(r, dst); err != nil {
		return err
	}

	if err := decodeHeader(r, dst); err != nil {
		return err
	}

	return validateStruct(dst)
}

func decodePath(r *http.Request, dst any) error {
	return decodeStruct(dst, func(tag, _ string) (string, bool) {
		v := chi.URLParam(r, tag)
		return v, v != ""
	}, nil, "path")
}

func decodeQuery(r *http.Request, dst any) error {
	q := r.URL.Query()
	return decodeStruct(dst, func(tag, _ string) (string, bool) {
		v := q.Get(tag)
		return v, v != ""
	}, func(tag string, fv reflect.Value) error {
		if fv.Kind() != reflect.Slice {
			return nil
		}

		raw := q[tag]
		if len(raw) == 0 {
			return nil
		}

		var vals []string
		for _, v := range raw {
			for part := range strings.SplitSeq(v, ",") {
				if s := strings.TrimSpace(part); s != "" {
					vals = append(vals, s)
				}
			}
		}

		return setSlice(fv, vals)
	}, "query")
}

func decodeHeader(r *http.Request, dst any) error {
	return decodeStruct(dst, func(tag, _ string) (string, bool) {
		v := r.Header.Get(tag)
		return v, v != ""
	}, nil, "header")
}

func decodeStruct(
	dst any,
	scalar func(tag, field string) (string, bool),
	sliceFn func(tag string, fv reflect.Value) error,
	tagName string,
) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return nil
	}

	rt := rv.Elem().Type()
	re := rv.Elem()

	for i := range rt.NumField() {
		field := rt.Field(i)
		fv := re.Field(i)

		if !fv.CanSet() {
			continue
		}

		tag := field.Tag.Get(tagName)
		if tag == "" || tag == "-" {
			continue
		}

		if fv.Kind() == reflect.Slice {
			if sliceFn != nil {
				if err := sliceFn(tag, fv); err != nil {
					return BadRequest("invalid "+tagName+" param: "+tag, err.Error())
				}
			}
			continue
		}

		raw, ok := scalar(tag, field.Name)
		if !ok {
			continue
		}

		if err := setScalar(fv, raw); err != nil {
			return BadRequest("invalid"+tagName+" param: "+tag, err.Error())
		}
	}

	return nil
}

func setScalar(fv reflect.Value, raw string) error {
	if fv.Kind() == reflect.Pointer {
		fv.Set(reflect.New(fv.Type().Elem()))
		fv = fv.Elem()
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(n)

	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	}

	return nil
}

func setSlice(fv reflect.Value, vals []string) error {
	elemType := fv.Type().Elem()
	slice := reflect.MakeSlice(fv.Type(), len(vals), len(vals))

	for i, v := range vals {
		elem := reflect.New(elemType).Elem()
		if err := setScalar(elem, v); err != nil {
			return err
		}
		slice.Index(i).Set(elem)
	}

	fv.Set(slice)
	return nil
}
