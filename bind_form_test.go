package kori

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBindForm_Scalars(t *testing.T) {
	type Form struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}

	body := strings.NewReader("name=Ada&age=36")
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var f Form
	if err := BindForm(req, &f); err != nil {
		t.Fatalf("BindForm: %v", err)
	}
	if f.Name != "Ada" || f.Age != 36 {
		t.Errorf("got %+v", f)
	}
}

func TestBindForm_SliceCommaExpansion(t *testing.T) {
	type Form struct {
		Tags []string `form:"tags"`
	}
	body := strings.NewReader("tags=a,b&tags=c, d ,")
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var f Form
	if err := BindForm(req, &f); err != nil {
		t.Fatalf("BindForm: %v", err)
	}
	want := []string{"a", "b", "c", "d"}
	if strings.Join(f.Tags, ",") != strings.Join(want, ",") {
		t.Errorf("Tags = %v, want %v", f.Tags, want)
	}
}

func TestBindForm_InvalidScalar400(t *testing.T) {
	type Form struct {
		Age int `form:"age"`
	}
	body := strings.NewReader("age=old")
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var f Form
	he := asHTTPError(t, BindForm(req, &f))
	if he.Status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", he.Status)
	}
}

func TestBindForm_Validation422(t *testing.T) {
	type Form struct {
		Name string `form:"name" validate:"required"`
	}
	body := strings.NewReader("other=x")
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var f Form
	he := asHTTPError(t, BindForm(req, &f))
	if he.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", he.Status)
	}
}

func newMultipart(t *testing.T, fields map[string]string, files map[string]string) (*http.Request, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	for field, content := range files {
		fw, err := mw.CreateFormFile(field, field+".txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req, buf.String()
}

func TestBindMultipart_ValuesAndSingleFile(t *testing.T) {
	type Form struct {
		Title string                `form:"title"`
		File  *multipart.FileHeader `form:"file"`
	}

	req, _ := newMultipart(t,
		map[string]string{"title": "report"},
		map[string]string{"file": "hello file"},
	)

	var f Form
	if err := BindMultipart(req, &f); err != nil {
		t.Fatalf("BindMultipart: %v", err)
	}
	if f.Title != "report" {
		t.Errorf("Title = %q, want report", f.Title)
	}
	if f.File == nil {
		t.Fatal("File header is nil")
	}
	if f.File.Size != int64(len("hello file")) {
		t.Errorf("File.Size = %d, want %d", f.File.Size, len("hello file"))
	}

	opened, err := f.File.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(opened)
	if buf.String() != "hello file" {
		t.Errorf("file content = %q, want %q", buf.String(), "hello file")
	}
}

func TestBindMultipart_MultipleFiles(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for i, content := range []string{"one", "two"} {
		fw, err := mw.CreateFormFile("docs", "doc"+string(rune('0'+i))+".txt")
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(content))
	}
	mw.Close()

	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	type Form struct {
		Docs []*multipart.FileHeader `form:"docs"`
	}
	var f Form
	if err := BindMultipart(req, &f); err != nil {
		t.Fatalf("BindMultipart: %v", err)
	}
	if len(f.Docs) != 2 {
		t.Fatalf("got %d files, want 2", len(f.Docs))
	}
}

func TestBindMultipart_FileTypeFieldSkippedInValueDecode(t *testing.T) {
	type Form struct {
		File *multipart.FileHeader `form:"file"`
		Name string                `form:"name"`
	}
	req, _ := newMultipart(t,
		map[string]string{"name": "n"},
		map[string]string{"file": "data"},
	)
	var f Form
	if err := BindMultipart(req, &f); err != nil {
		t.Fatalf("BindMultipart: %v", err)
	}
	if f.Name != "n" || f.File == nil {
		t.Errorf("got %+v", f)
	}
}

func TestBindMultipart_BadContentType(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "text/plain")

	type Form struct {
		Name string `form:"name"`
	}
	var f Form
	he := asHTTPError(t, BindMultipart(req, &f))
	if he.Status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", he.Status)
	}
}

func TestSetMaxMultipartMemory(t *testing.T) {
	t.Cleanup(func() { SetMaxMultipartMemory(32 << 20) })
	SetMaxMultipartMemory(1)
	if got := multipartMaxMemory.Load(); got != 1 {
		t.Errorf("multipartMaxMemory = %d, want 1", got)
	}
}

func TestIsFileType(t *testing.T) {
	if !isFileType(fileHeaderPtrType) {
		t.Error("*multipart.FileHeader should be a file type")
	}
	if !isFileType(fileHeaderSliceType) {
		t.Error("[]*multipart.FileHeader should be a file type")
	}
	type notFile struct{}
	if isFileType(fileHeaderPtrType.Elem()) {
		t.Error("multipart.FileHeader (value) should not be a file type")
	}
	_ = notFile{}
}
