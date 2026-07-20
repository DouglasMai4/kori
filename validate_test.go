package kori

import (
	"net/http"
	"strings"
	"testing"
)

func TestValidateStruct_Passes(t *testing.T) {
	type T struct {
		Name string `json:"name" validate:"required"`
	}
	if err := validateStruct(&T{Name: "ok"}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateStruct_ReturnsDetails(t *testing.T) {
	type T struct {
		Name  string `json:"name"  validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	err := validateStruct(&T{Email: "bad"})
	he := asHTTPError(t, err)

	if he.Status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", he.Status)
	}
	if he.Message != "validation failed" {
		t.Errorf("message = %q", he.Message)
	}

	details, ok := he.Details.([]ValidationDetail)
	if !ok {
		t.Fatalf("Details type = %T, want []ValidationDetail", he.Details)
	}
	if len(details) != 2 {
		t.Fatalf("got %d details, want 2", len(details))
	}

	byField := map[string]ValidationDetail{}
	for _, d := range details {
		byField[d.Field] = d
	}
	if _, ok := byField["name"]; !ok {
		t.Errorf("expected a detail for json field 'name', got %v", byField)
	}
	if _, ok := byField["email"]; !ok {
		t.Errorf("expected a detail for json field 'email', got %v", byField)
	}
}

func TestValidateStruct_TagNamePriority(t *testing.T) {
	type T struct {
		Page int `query:"page" validate:"min=1"`
	}
	err := validateStruct(&T{Page: 0})
	he := asHTTPError(t, err)
	details := he.Details.([]ValidationDetail)
	if details[0].Field != "page" {
		t.Errorf("Field = %q, want page (from query tag)", details[0].Field)
	}
}

func TestValidationMessages(t *testing.T) {
	cases := []struct {
		name        string
		validate    string
		value       any
		wantTag     string
		wantSnippet string
	}{
		{"required", "required", struct {
			F string `json:"f" validate:"required"`
		}{}, "required", "f is required"},
		{"min_string", "min=3", struct {
			F string `json:"f" validate:"min=3"`
		}{F: "ab"}, "min", "must be at least 3"},
		{"max_string", "max=2", struct {
			F string `json:"f" validate:"max=2"`
		}{F: "abcd"}, "max", "must be at most 2"},
		{"email", "email", struct {
			F string `json:"f" validate:"email"`
		}{F: "nope"}, "email", "valid email address"},
		{"oneof", "oneof=a b", struct {
			F string `json:"f" validate:"oneof=a b"`
		}{F: "c"}, "oneof", "must be one of: a b"},
		{"url", "url", struct {
			F string `json:"f" validate:"url"`
		}{F: "notaurl"}, "url", "valid URL"},
		{"gt", "gt=5", struct {
			F int `json:"f" validate:"gt=5"`
		}{F: 5}, "gt", "greater than 5"},
		{"gte", "gte=5", struct {
			F int `json:"f" validate:"gte=5"`
		}{F: 4}, "gte", "greater than or equal to 5"},
		{"lt", "lt=5", struct {
			F int `json:"f" validate:"lt=5"`
		}{F: 5}, "lt", "less than 5"},
		{"lte", "lte=5", struct {
			F int `json:"f" validate:"lte=5"`
		}{F: 6}, "lte", "less than or equal to 5"},
		{"len", "len=4", struct {
			F string `json:"f" validate:"len=4"`
		}{F: "ab"}, "len", "exactly 4 characters"},
		{"uuid", "uuid", struct {
			F string `json:"f" validate:"uuid"`
		}{F: "x"}, "uuid", "valid UUID"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateStruct(tc.value)
			he := asHTTPError(t, err)
			details := he.Details.([]ValidationDetail)
			if len(details) == 0 {
				t.Fatal("no validation details produced")
			}
			d := details[0]
			if d.Tag != tc.wantTag {
				t.Errorf("Tag = %q, want %q", d.Tag, tc.wantTag)
			}
			if !strings.Contains(d.Message, tc.wantSnippet) {
				t.Errorf("Message = %q, want it to contain %q", d.Message, tc.wantSnippet)
			}
		})
	}
}

func TestValidationMessage_UnknownTagFallsBackToLibraryError(t *testing.T) {
	type T struct {
		F string `json:"f" validate:"alpha"`
	}
	err := validateStruct(&T{F: "123"})
	he := asHTTPError(t, err)
	details := he.Details.([]ValidationDetail)
	if details[0].Message == "" {
		t.Error("fallback message should not be empty")
	}
	if details[0].Tag != "alpha" {
		t.Errorf("Tag = %q, want alpha", details[0].Tag)
	}
}

func TestGetValidator_Singleton(t *testing.T) {
	first := getValidator()
	second := getValidator()
	if first != second {
		t.Error("getValidator must return the same cached instance")
	}
}
