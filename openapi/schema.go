package openapi

import (
	"maps"
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
	if t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}

	s := wellKnownSchema(t)

	if s == nil {
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
	} // end if s == nil

	if nullable && s != nil && s.Ref == "" {
		if typStr, ok := s.Type.(string); ok && typStr != "" {
			s.Type = []string{typStr, "null"}
		}
	}

	return s
}

func (r *registry) structSchema(t reflect.Type) *Schema {
	name := t.Name()
	if name == "" {
		return r.buildStructFields(t)
	}

	safeName := sanitizeSchemaName(name)

	r.mu.Lock()
	if r.inProgress[t] {
		r.mu.Unlock()
		return &Schema{Ref: "#/components/schemas/" + safeName}
	}
	if _, already := r.schemas[safeName]; already {
		r.mu.Unlock()
		return &Schema{Ref: "#/components/schemas/" + safeName}
	}
	r.inProgress[t] = true
	r.mu.Unlock()

	s := r.buildStructFields(t)

	r.mu.Lock()
	r.schemas[safeName] = s
	delete(r.inProgress, t)
	r.mu.Unlock()

	return &Schema{Ref: "#/components/schemas/" + safeName}
}

func (r *registry) buildStructFields(t reflect.Type) *Schema {
	s := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for field := range t.Fields() {
		field := field

		if !field.IsExported() && !field.Anonymous {
			continue
		}

		if field.Anonymous {
			ft := field.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				embedded := r.buildStructFields(ft)
				maps.Copy(s.Properties, embedded.Properties)
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

	for rule := range strings.SplitSeq(tag, ",") {
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
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return f.Name
	}
	return name
}

func isRequired(f reflect.StructField) bool {
	if f.Type.Kind() == reflect.Pointer {
		return false
	}
	for rule := range strings.SplitSeq(f.Tag.Get("validate"), ",") {
		if strings.TrimSpace(rule) == "required" {
			return true
		}
	}
	return false
}

func baseKind(t reflect.Type) reflect.Kind {
	for t.Kind() == reflect.Pointer {
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

// wellKnownSchemas maps "<pkg-path>.<type-name>" to a fixed OpenAPI schema
// for types whose Go struct representation does not reflect their wire format.
// Keys follow the pattern returned by reflect.Type.PkgPath() + "." + Name().
var wellKnownSchemas = map[string]*Schema{
	// Standard library
	"time.Time":                {Type: "string", Format: "date-time"},
	"net/url.URL":              {Type: "string", Format: "uri"},
	"net.IP":                   {Type: "string", Format: "ipv4"},
	"encoding/json.RawMessage": {},

	// github.com/google/uuid
	"github.com/google/uuid.UUID": {Type: "string", Format: "uuid"},
}

// wellKnownSchema returns a fresh copy of the pre-defined schema for t if one
// exists, or nil if t should be reflected normally.
func wellKnownSchema(t reflect.Type) *Schema {
	key := t.PkgPath() + "." + t.Name()
	if tmpl, ok := wellKnownSchemas[key]; ok {
		s := *tmpl
		return &s
	}
	return nil
}

// sanitizeSchemaName converts a Go type name into a valid OpenAPI component
// name (matching ^[a-zA-Z0-9._-]+$). Generic instantiations such as
// "SuccessResponse[pkg.HomeResponse]" become "SuccessResponse_HomeResponse".
func sanitizeSchemaName(name string) string {
	if !strings.ContainsAny(name, "[]") {
		return name
	}
	var b strings.Builder
	i, n := 0, len(name)
	for i < n {
		c := name[i]
		switch c {
		case '[', ',':
			b.WriteByte('_')
			i++
			depth, j := 0, i
		paramScan:
			for j < n {
				switch name[j] {
				case '[':
					depth++
				case ']':
					if depth == 0 {
						break paramScan
					}
					depth--
				case ',':
					if depth == 0 {
						break paramScan
					}
				}
				j++
			}
			param := strings.TrimLeft(name[i:j], "[]* ")
			i = j
			typeEnd := strings.IndexByte(param, '[')
			if typeEnd < 0 {
				typeEnd = len(param)
			}
			typePart := param[:typeEnd]
			if dot := strings.LastIndex(typePart, "."); dot >= 0 {
				typePart = typePart[dot+1:]
			}
			b.WriteString(typePart)
			// Recursively sanitize any nested generic portion.
			if typeEnd < len(param) {
				if inner := sanitizeSchemaName(param[typeEnd:]); inner != "" {
					b.WriteByte('_')
					b.WriteString(inner)
				}
			}
		case ']', ' ', '*':
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	s := b.String()
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return strings.Trim(s, "_")
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
