package openapi

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"
)

func schemaForType(v any) *Schema {
	return newRegistry().schemaFor(reflect.TypeOf(v))
}

func TestSchemaFor_Primitives(t *testing.T) {
	cases := []struct {
		val        any
		wantType   string
		wantFormat string
	}{
		{"", "string", ""},
		{int32(0), "integer", "int32"},
		{int(0), "integer", "int32"},
		{int64(0), "integer", "int64"},
		{uint(0), "integer", "int64"},
		{float32(0), "number", "float"},
		{float64(0), "number", "double"},
		{true, "boolean", ""},
	}
	for _, tc := range cases {
		s := schemaForType(tc.val)
		typ, _ := s.Type.(string)
		if typ != tc.wantType {
			t.Errorf("%T: Type = %v, want %q", tc.val, s.Type, tc.wantType)
		}
		if s.Format != tc.wantFormat {
			t.Errorf("%T: Format = %q, want %q", tc.val, s.Format, tc.wantFormat)
		}
	}
}

func TestSchemaFor_NullablePointer(t *testing.T) {
	var p *string
	s := newRegistry().schemaFor(reflect.TypeOf(p))
	types, ok := s.Type.([]string)
	if !ok {
		t.Fatalf("Type = %v (%T), want []string", s.Type, s.Type)
	}
	if len(types) != 2 || types[0] != "string" || types[1] != "null" {
		t.Errorf("Type = %v, want [string null]", types)
	}
}

func TestSchemaFor_Slice(t *testing.T) {
	s := schemaForType([]int{})
	if s.Type != "array" {
		t.Fatalf("Type = %v, want array", s.Type)
	}
	if s.Items == nil || s.Items.Type != "integer" {
		t.Errorf("Items = %+v, want integer", s.Items)
	}
}

func TestSchemaFor_Map(t *testing.T) {
	s := schemaForType(map[string]int{})
	if s.Type != "object" {
		t.Errorf("Type = %v, want object", s.Type)
	}
	if s.AdditionalProperties != true {
		t.Errorf("AdditionalProperties = %v, want true", s.AdditionalProperties)
	}
}

func TestSchemaFor_WellKnownTypes(t *testing.T) {
	tm := schemaForType(time.Time{})
	if tm.Type != "string" || tm.Format != "date-time" {
		t.Errorf("time.Time schema = %+v, want string/date-time", tm)
	}
	raw := schemaForType(json.RawMessage{})

	if raw.Type != nil {
		t.Errorf("json.RawMessage Type = %v, want nil (any)", raw.Type)
	}
}

func TestSchemaFor_WellKnownReturnsFreshCopy(t *testing.T) {
	a := schemaForType(time.Time{})
	a.Description = "mutated"
	b := schemaForType(time.Time{})
	if b.Description != "" {
		t.Error("wellKnownSchema template was mutated across calls")
	}
}

type Address struct {
	Street string `json:"street" validate:"required"`
	City   string `json:"city"`
}

type Person struct {
	Name    string  `json:"name" validate:"required" doc:"Full name" example:"Ada"`
	Age     int     `json:"age" validate:"min=0,max=130"`
	Email   string  `json:"email" validate:"email"`
	Home    Address `json:"home"`
	Nick    *string `json:"nick"`
	Ignored string  `json:"-"`
	private string
	Tags    []string `json:"tags"`
}

func TestStructSchema_RefRegistration(t *testing.T) {
	reg := newRegistry()
	s := reg.schemaFor(reflect.TypeFor[Person]())

	if s.Ref != "#/components/schemas/Person" {
		t.Fatalf("Ref = %q, want #/components/schemas/Person", s.Ref)
	}
	if _, ok := reg.schemas["Person"]; !ok {
		t.Fatal("Person not registered")
	}
	if _, ok := reg.schemas["Address"]; !ok {
		t.Fatal("nested Address not registered")
	}

	person := reg.schemas["Person"]
	if person.Type != "object" {
		t.Errorf("Person.Type = %v, want object", person.Type)
	}

	if _, ok := person.Properties["name"]; !ok {
		t.Error("missing 'name' property")
	}
	if _, ok := person.Properties["Ignored"]; ok {
		t.Error("json:\"-\" field must be excluded")
	}
	if _, ok := person.Properties["private"]; ok {
		t.Error("unexported field must be excluded")
	}

	if person.Properties["name"].Description != "Full name" {
		t.Errorf("name.Description = %q", person.Properties["name"].Description)
	}
	if person.Properties["name"].Example != "Ada" {
		t.Errorf("name.Example = %v, want Ada", person.Properties["name"].Example)
	}

	if !contains(person.Required, "name") {
		t.Errorf("Required = %v, want to contain name", person.Required)
	}
	if contains(person.Required, "nick") {
		t.Error("pointer field nick must not be required")
	}
}

func TestStructSchema_NumericConstraints(t *testing.T) {
	reg := newRegistry()
	reg.schemaFor(reflect.TypeFor[Person]())
	age := reg.schemas["Person"].Properties["age"]

	if age.Minimum == nil || *age.Minimum != 0 {
		t.Errorf("age.Minimum = %v, want 0", age.Minimum)
	}
	if age.Maximum == nil || *age.Maximum != 130 {
		t.Errorf("age.Maximum = %v, want 130", age.Maximum)
	}
}

func TestStructSchema_EmailFormat(t *testing.T) {
	reg := newRegistry()
	reg.schemaFor(reflect.TypeFor[Person]())
	email := reg.schemas["Person"].Properties["email"]
	if email.Format != "email" {
		t.Errorf("email.Format = %q, want email", email.Format)
	}
}

type selfRef struct {
	Name     string     `json:"name"`
	Children []*selfRef `json:"children"`
}

func TestStructSchema_RecursiveTypeTerminates(t *testing.T) {
	reg := newRegistry()
	done := make(chan *Schema, 1)
	go func() { done <- reg.schemaFor(reflect.TypeFor[selfRef]()) }()

	select {
	case s := <-done:
		if s.Ref != "#/components/schemas/selfRef" {
			t.Errorf("Ref = %q", s.Ref)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("recursive schema generation did not terminate (cycle guard failed)")
	}

	self := reg.schemas["selfRef"]
	items := self.Properties["children"].Items
	if items == nil || items.Ref != "#/components/schemas/selfRef" {
		t.Errorf("children.Items = %+v, want $ref to selfRef", items)
	}
}

type Base struct {
	ID string `json:"id" validate:"required"`
}

type Derived struct {
	Base
	Extra string `json:"extra"`
}

func TestStructSchema_EmbeddedFieldsFlattened(t *testing.T) {
	reg := newRegistry()
	reg.schemaFor(reflect.TypeFor[Derived]())
	d := reg.schemas["Derived"]
	if _, ok := d.Properties["id"]; !ok {
		t.Error("embedded field 'id' not flattened into Derived")
	}
	if _, ok := d.Properties["extra"]; !ok {
		t.Error("missing own field 'extra'")
	}
	if !contains(d.Required, "id") {
		t.Errorf("Required = %v, want to include promoted 'id'", d.Required)
	}
}

type unexportedBase struct {
	Slug string `json:"slug" validate:"required"`
}

type withUnexportedEmbed struct {
	unexportedBase
	Title string `json:"title"`
}

func TestStructSchema_UnexportedEmbeddedFieldsPromoted(t *testing.T) {
	reg := newRegistry()
	reg.schemaFor(reflect.TypeFor[withUnexportedEmbed]())
	s := reg.schemas["withUnexportedEmbed"]

	if _, ok := s.Properties["slug"]; !ok {
		t.Error("promoted field 'slug' from unexported embedded type not in schema")
	}
	if !contains(s.Required, "slug") {
		t.Errorf("Required = %v, want to include promoted 'slug'", s.Required)
	}
}

func TestEnrichFromValidate(t *testing.T) {
	t.Run("string min/max become length", func(t *testing.T) {
		s := &Schema{Type: "string"}
		enrichFromValidate(s, "min=2,max=8", reflect.String)
		if s.MinLength == nil || *s.MinLength != 2 {
			t.Errorf("MinLength = %v, want 2", s.MinLength)
		}
		if s.MaxLength == nil || *s.MaxLength != 8 {
			t.Errorf("MaxLength = %v, want 8", s.MaxLength)
		}
	})

	t.Run("numeric min/max become minimum/maximum", func(t *testing.T) {
		s := &Schema{Type: "integer"}
		enrichFromValidate(s, "min=1,max=5", reflect.Int)
		if s.Minimum == nil || *s.Minimum != 1 {
			t.Errorf("Minimum = %v, want 1", s.Minimum)
		}
		if s.Maximum == nil || *s.Maximum != 5 {
			t.Errorf("Maximum = %v, want 5", s.Maximum)
		}
	})

	t.Run("gt/lt become exclusive bounds", func(t *testing.T) {
		s := &Schema{Type: "number"}
		enrichFromValidate(s, "gt=0,lt=100", reflect.Float64)
		if s.ExclusiveMinimum == nil || *s.ExclusiveMinimum != 0 {
			t.Errorf("ExclusiveMinimum = %v, want 0", s.ExclusiveMinimum)
		}
		if s.ExclusiveMaximum == nil || *s.ExclusiveMaximum != 100 {
			t.Errorf("ExclusiveMaximum = %v, want 100", s.ExclusiveMaximum)
		}
	})

	t.Run("len sets both length bounds", func(t *testing.T) {
		s := &Schema{Type: "string"}
		enrichFromValidate(s, "len=6", reflect.String)
		if s.MinLength == nil || *s.MinLength != 6 || s.MaxLength == nil || *s.MaxLength != 6 {
			t.Errorf("len should pin MinLength=MaxLength=6, got min=%v max=%v", s.MinLength, s.MaxLength)
		}
	})

	t.Run("oneof becomes enum", func(t *testing.T) {
		s := &Schema{Type: "string"}
		enrichFromValidate(s, "oneof=red green blue", reflect.String)
		if len(s.Enum) != 3 || s.Enum[0] != "red" || s.Enum[2] != "blue" {
			t.Errorf("Enum = %v, want [red green blue]", s.Enum)
		}
	})

	t.Run("format rules", func(t *testing.T) {
		for rule, want := range map[string]string{
			"email":    "email",
			"uuid":     "uuid",
			"url":      "uri",
			"datetime": "date-time",
		} {
			s := &Schema{Type: "string"}
			enrichFromValidate(s, rule, reflect.String)
			if s.Format != want {
				t.Errorf("rule %q: Format = %q, want %q", rule, s.Format, want)
			}
		}
	})

	t.Run("empty tag is a noop", func(t *testing.T) {
		s := &Schema{Type: "string"}
		enrichFromValidate(s, "", reflect.String)
		if s.MinLength != nil || s.Format != "" {
			t.Error("empty validate tag should not modify schema")
		}
	})
}

func TestSanitizeSchemaName(t *testing.T) {
	cases := map[string]string{
		"User":                              "User",
		"SuccessResponse[pkg.HomeResponse]": "SuccessResponse_HomeResponse",
		"List[github.com/x/y.Item]":         "List_Item",
		"Page[*pkg.Thing]":                  "Page_Thing",
		"Map[string,pkg.Val]":               "Map_string_Val",
		"Wrap[Outer[pkg.Inner]]":            "Wrap_Outer_Inner",
		"Paginated[[]github.com/x/y.Item]":  "Paginated_Item",
		"Paginated[[]*github.com/x/y.Item]": "Paginated_Item",
	}
	for in, want := range cases {
		if got := sanitizeSchemaName(in); got != want {
			t.Errorf("sanitizeSchemaName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseExampleValue(t *testing.T) {
	if v := parseExampleValue("42", reflect.Int); v != int64(42) {
		t.Errorf("int example = %v (%T), want int64(42)", v, v)
	}
	if v := parseExampleValue("3.14", reflect.Float64); v != 3.14 {
		t.Errorf("float example = %v, want 3.14", v)
	}
	if v := parseExampleValue("true", reflect.Bool); v != true {
		t.Errorf("bool example = %v, want true", v)
	}
	if v := parseExampleValue("hello", reflect.String); v != "hello" {
		t.Errorf("string example = %v, want hello", v)
	}
	if v := parseExampleValue("N/A", reflect.Int); v != "N/A" {
		t.Errorf("invalid numeric example = %v, want raw string", v)
	}
}

func TestIsRequired(t *testing.T) {
	typ := reflect.TypeOf(struct {
		A string  `validate:"required"`
		B *string `validate:"required"`
		C string  `validate:"min=1"`
	}{})
	if !isRequired(typ.Field(0)) {
		t.Error("A should be required")
	}
	if isRequired(typ.Field(1)) {
		t.Error("pointer B must never be required")
	}
	if isRequired(typ.Field(2)) {
		t.Error("C without 'required' rule should not be required")
	}
}

func contains(s []string, v string) bool {
	return slices.Contains(s, v)
}
