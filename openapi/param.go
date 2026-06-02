package openapi

import (
	"reflect"
)

func extractParams(dst any, reg *registry) []Parameter {
	if dst == nil {
		return nil
	}

	t := reflect.TypeOf(dst)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	var params []Parameter

	for i := range t.NumField() {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		if field.Anonymous {
			ft := field.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				params = append(params, extractParams(reflect.New(ft).Elem().Interface(), reg)...)
			}
			continue
		}

		var in, name string
		var required bool

		switch {
		case field.Tag.Get("path") != "":
			in = "path"
			name = field.Tag.Get("path")
			required = true

		case field.Tag.Get("query") != "":
			in = "query"
			name = field.Tag.Get("query")
			required = isRequired(field)

		case field.Tag.Get("header") != "":
			in = "header"
			name = field.Tag.Get("header")
			required = isRequired(field)

		default:
			continue
		}

		s := reg.schemaFor(field.Type)
		if s != nil && s.Ref == "" {
			if doc := field.Tag.Get("doc"); doc != "" {
				s.Description = doc
			}
			if ex := field.Tag.Get("example"); ex != "" {
				s.Example = parseExampleValue(ex, baseKind(field.Type))
			}
			enrichFromValidate(s, field.Tag.Get("validate"), baseKind(field.Type))
		}

		p := Parameter{
			Name:     name,
			In:       in,
			Required: required,
			Schema:   s,
		}

		if doc := field.Tag.Get("doc"); doc != "" {
			p.Description = doc
		}
		if ex := field.Tag.Get("example"); ex != "" {
			p.Example = parseExampleValue(ex, baseKind(field.Type))
		}

		if in == "query" && field.Type.Kind() == reflect.Slice {
		}

		params = append(params, p)
	}

	return params
}
