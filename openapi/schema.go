package openapi

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type registry struct {
	mu         sync.Mutex
	schemas    map[string]*Schema
	inProgress map[reflect.Type]bool
}

func newRegistry() *registry {
	return &registry{
		schemas:    make(map[string]*Schema),
		inProgress: make(map[reflect.Type]bool),
	}
}

func (r *registry) schemaFor(t reflect.Type) *Schema {
	nullable := false
	if t.Kind() == reflect.Ptr {
		nullable = true
		t = t.Elem()
	}

	var s *Schema

	switch t.Kind() {
	case reflect.String:
		s = &Schema{Type: "string"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		s = &Schema{Type: "integer", Format: "int32"}
	case reflect.Int64:
		s = &Schema{Type: "integer", Format: "int64"}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s = &Schema{Type: "integer", Format: "int64"}

	case reflect.Float32:
		s = &Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		s = &Schema{Type: "number", Format: "double"}

	case reflect.Bool:
		s = &Schema{Type: "boolean"}

	case reflect.Slice:
		items := r.schemaFor(t.Elem())
		s = &Schema{Type: "array", Items: items}

	case reflect.Map:
		s = &Schema{Type: "object", AdditionalProperties: true}

	case reflect.Struct:
		s = r.structSchema(t)

	case reflect.Interface:
		s = &Schema{}

	default:
		s = &Schema{Type: "string"}
	}

	if nullable && s != nil && s.Ref == "" {
		if typStr, ok := s.Type.(string); ok && typStr != "" {
			s.Type = []string{typStr, "null"}
		}
	}

	return s
}

func (r *registry) structSchema(t reflect.Type) *Schema {
	name := t.Name()

	if name != "" {
		r.mu.Lock()
		if r.inProgress[t] {
			r.mu.Unlock()
			return &Schema{Ref: "#/components/schemas/" + name}
		}
		if _, already := r.schemas[name]; already {
			r.mu.Unlock()
			return &Schema{Ref: "#/components/schemas/" + name}
		}
		r.inProgress[t] = true
		r.mu.Unlock()

		s := r.buildStructFields(t)

		r.mu.Lock()
		r.schemas[name] = s
		delete(r.inProgress, t)
		r.mu.Unlock()

		return &Schema{Ref: "#/components/schemas/" + name}
	}

	return r.buildStructFields(t)
}

func (r *registry) buildStructFields(t reflect.Type) *Schema {
	s := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

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
				embedded := r.buildStructFields(ft)
				for k, v := range embedded.Properties {
					s.Properties[k] = v
				}
				s.Required = append(s.Required, embedded.Required...)
			}
			continue
		}

		jsonName := jsonFieldName(field)
		if jsonName == "-" {
			continue
		}

		propSchema := r.schemaFor(field.Type)

		applyFieldTags(propSchema, field)

		s.Properties[jsonName] = propSchema

		if isRequired(field) {
			s.Required = append(s.Required, jsonName)
		}
	}

	if len(s.Properties) == 0 {
		s.Properties = nil
	}
	if len(s.Required) == 0 {
		s.Required = nil
	}

	return s
}

func applyFieldTags(s *Schema, field reflect.StructField) {
	if s.Ref != "" {
		return
	}

	if doc := field.Tag.Get("doc"); doc != "" {
		s.Description = doc
	}

	if ex := field.Tag.Get("example"); ex != "" {
		s.Example = parseExampleValue(ex, baseKind(field.Type))
	}

	enrichFromValidate(s, field.Tag.Get("validate"), baseKind(field.Type))
}

func enrichFromValidate(s *Schema, tag string, kind reflect.Kind) {
	if tag == "" {
		return
	}

	for _, rule := range strings.Split(tag, ",") {
		rule = strings.TrimSpace(rule)
		switch {
		case rule == "email":
			s.Format = "email"
		case rule == "uuid" || rule == "uuid4" || rule == "uuid3" || rule == "uuid5":
			s.Format = "uuid"
		case rule == "url" || rule == "uri" || rule == "http_url":
			s.Format = "uri"
		case rule == "datetime":
			s.Format = "date-time"

		case strings.HasPrefix(rule, "min="):
			n, err := strconv.ParseFloat(rule[4:], 64)
			if err != nil {
				continue
			}
			if isStringKind(kind) {
				v := int(n)
				s.MinLength = &v
			} else if isNumericKind(kind) {
				v := float64(n)
				s.Minimum = &v
			}

		case strings.HasPrefix(rule, "max="):
			n, err := strconv.ParseFloat(rule[4:], 64)
			if err != nil {
				continue
			}
			if isStringKind(kind) {
				v := int(n)
				s.MaxLength = &v
			} else if isNumericKind(kind) {
				v := float64(n)
				s.Maximum = &v
			}

		case strings.HasPrefix(rule, "gt="):
			n, err := strconv.ParseFloat(rule[3:], 64)
			if err != nil {
				continue
			}
			if isNumericKind(kind) {
				v := n
				s.ExclusiveMinimum = &v
			}

		case strings.HasPrefix(rule, "lt="):
			n, err := strconv.ParseFloat(rule[3:], 64)
			if err != nil {
				continue
			}
			if isNumericKind(kind) {
				v := n
				s.ExclusiveMaximum = &v
			}

		case strings.HasPrefix(rule, "len="):
			n, err := strconv.Atoi(rule[4:])
			if err != nil {
				continue
			}
			if isStringKind(kind) {
				s.MinLength = &n
				maxN := n
				s.MaxLength = &maxN
			}

		case strings.HasPrefix(rule, "oneof="):
			vals := strings.Fields(rule[6:])
			s.Enum = make([]any, len(vals))
			for i, v := range vals {
				s.Enum[i] = v
			}
		}
	}
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return f.Name
	}
	return name
}

func isRequired(f reflect.StructField) bool {
	if f.Type.Kind() == reflect.Ptr {
		return false
	}
	for _, rule := range strings.Split(f.Tag.Get("validate"), ",") {
		if strings.TrimSpace(rule) == "required" {
			return true
		}
	}
	return false
}

func baseKind(t reflect.Type) reflect.Kind {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind()
}

func isStringKind(k reflect.Kind) bool {
	return k == reflect.String
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func parseExampleValue(raw string, kind reflect.Kind) any {
	switch {
	case isNumericKind(kind):
		if n, err := strconv.ParseFloat(raw, 64); err == nil {
			if n == float64(int64(n)) {
				return int64(n)
			}
			return n
		}
	case kind == reflect.Bool:
		if b, err := strconv.ParseBool(raw); err == nil {
			return b
		}
	}
	return raw
}
