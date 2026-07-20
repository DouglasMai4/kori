package kori

import (
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
)

var multipartMaxMemory atomic.Int64

func init() {
	multipartMaxMemory.Store(32 << 20)
}

// SetMaxMultipartMemory sets the maximum memory size for multipart form parsing.
// Default is 32 MB.
func SetMaxMultipartMemory(n int64) {
	multipartMaxMemory.Store(n)
}

// BindForm decodes form values from r.PostForm into dst and validates it.
// dst must be a pointer to a struct with `form` tags.
func BindForm(r *http.Request, dst any) error {
	if err := r.ParseForm(); err != nil {
		return BadRequest("cannot parse form body", err.Error())
	}
	if err := decodeFormValues(r.PostForm, dst); err != nil {
		return err
	}
	return validateStruct(dst)
}

// BindMultipart decodes multipart form data (values and files) into dst and validates it.
// dst must be a pointer to a struct with `form` tags.
// File fields must be of type *multipart.FileHeader or []*multipart.FileHeader.
func BindMultipart(r *http.Request, dst any) error {
	if err := r.ParseMultipartForm(multipartMaxMemory.Load()); err != nil {
		return BadRequest("cannot parse multipart form", err.Error())
	}

	mf := r.MultipartForm
	if err := decodeFormValues(mf.Value, dst); err != nil {
		return err
	}
	if err := decodeFormFiles(mf.File, dst); err != nil {
		return err
	}

	return validateStruct(dst)
}

func decodeFormValues(values map[string][]string, dst any) error {
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

		tag := field.Tag.Get("form")
		if tag == "" || tag == "-" {
			continue
		}

		if isFileType(fv.Type()) {
			continue
		}

		vals, ok := values[tag]
		if !ok || len(vals) == 0 {
			continue
		}

		if fv.Kind() == reflect.Slice {
			var expanded []string
			for _, v := range vals {
				for part := range strings.SplitSeq(v, ",") {
					if s := strings.TrimSpace(part); s != "" {
						expanded = append(expanded, s)
					}
				}
			}

			if err := setSlice(fv, expanded); err != nil {
				return BadRequest("invalid form field: "+tag, err.Error())
			}
			continue
		}

		if err := setScalar(fv, vals[0]); err != nil {
			return BadRequest("invalid form field: "+tag, err.Error())
		}
	}
	return nil
}

func decodeFormFiles(files map[string][]*multipart.FileHeader, dst any) error {
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

		tag := field.Tag.Get("form")
		if tag == "" || tag == "-" {
			continue
		}

		hdrs, ok := files[tag]
		if !ok || len(hdrs) == 0 {
			continue
		}

		switch fv.Type() {
		case fileHeaderPtrType:
			fv.Set(reflect.ValueOf(hdrs[0]))

		case fileHeaderSliceType:
			fv.Set(reflect.ValueOf(hdrs))
		}
	}

	return nil
}

var (
	fileHeaderPtrType   = reflect.TypeFor[*multipart.FileHeader]()
	fileHeaderSliceType = reflect.TypeFor[[]*multipart.FileHeader]()
)

func isFileType(t reflect.Type) bool {
	return t == fileHeaderPtrType || t == fileHeaderSliceType
}
