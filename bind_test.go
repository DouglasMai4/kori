package kori

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func asHTTPError(t *testing.T, err error) *HTTPError {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("error is not *HTTPError: %T (%v)", err, err)
	}
	return he
}

func TestBindQuery_Scalars(t *testing.T) {
	type Params struct {
		Name   string  `query:"name"`
		Age    int     `query:"age"`
		Score  float64 `query:"score"`
		Active bool    `query:"active"`
	}

	req := httptest.NewRequest("GET", "/?name=jane&age=30&score=9.5&active=true", nil)
	var p Params
	if err := BindQuery(req, &p); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}

	if p.Name != "jane" || p.Age != 30 || p.Score != 9.5 || p.Active != true {
		t.Errorf("got %+v", p)
	}
}

func TestBindQuery_MissingLeavesZeroValue(t *testing.T) {
	type Params struct {
		Name string `query:"name"`
		Age  int    `query:"age"`
	}

	req := httptest.NewRequest("GET", "/?name=jane", nil)
	var p Params
	if err := BindQuery(req, &p); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}
	if p.Age != 0 {
		t.Errorf("Age = %d, want 0 (missing param must not error)", p.Age)
	}
}

func TestBindQuery_InvalidIntReturns400(t *testing.T) {
	type Params struct {
		Age int `query:"age"`
	}
	req := httptest.NewRequest("GET", "/?age=notanumber", nil)
	var p Params
	he := asHTTPError(t, BindQuery(req, &p))
	if he.Status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", he.Status)
	}
	if !strings.Contains(he.Message, "age") {
		t.Errorf("message = %q, want it to mention the field", he.Message)
	}
}

func TestBindQuery_Slice_CommaAndRepeat(t *testing.T) {
	type Params struct {
		Tags []string `query:"tags"`
		Nums []int    `query:"nums"`
	}

	req := httptest.NewRequest("GET", "/?tags=a,b&tags=c,,d&nums=1,2&nums=3", nil)
	var p Params
	if err := BindQuery(req, &p); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}

	wantTags := []string{"a", "b", "c", "d"}
	if strings.Join(p.Tags, ",") != strings.Join(wantTags, ",") {
		t.Errorf("Tags = %v, want %v", p.Tags, wantTags)
	}
	if len(p.Nums) != 3 || p.Nums[0] != 1 || p.Nums[1] != 2 || p.Nums[2] != 3 {
		t.Errorf("Nums = %v, want [1 2 3]", p.Nums)
	}
}

func TestBindQuery_EmptySliceStaysNil(t *testing.T) {
	type Params struct {
		Tags []string `query:"tags"`
	}
	req := httptest.NewRequest("GET", "/", nil)
	var p Params
	if err := BindQuery(req, &p); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}
	if p.Tags != nil {
		t.Errorf("Tags = %v, want nil for absent param", p.Tags)
	}
}

func TestBindQuery_SliceInvalidElementReturns400(t *testing.T) {
	type Params struct {
		Nums []int `query:"nums"`
	}
	req := httptest.NewRequest("GET", "/?nums=1,x,3", nil)
	var p Params
	he := asHTTPError(t, BindQuery(req, &p))
	if he.Status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", he.Status)
	}
}

func TestBindQuery_Validation422(t *testing.T) {
	type Params struct {
		Page int `query:"page" validate:"min=1"`
	}
	req := httptest.NewRequest("GET", "/?page=0", nil)
	var p Params
	he := asHTTPError(t, BindQuery(req, &p))
	if he.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", he.Status)
	}
}

func TestBindPath_Scalars(t *testing.T) {
	type Params struct {
		ID   int    `path:"id"`
		Slug string `path:"slug"`
	}

	r := chi.NewRouter()
	var captured Params
	var bindErr error
	GET(r, "/items/{id}/{slug}", func(w http.ResponseWriter, req *http.Request) error {
		bindErr = BindPath(req, &captured)
		return NoContent(w)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/items/7/hello", nil))

	if bindErr != nil {
		t.Fatalf("BindPath: %v", bindErr)
	}
	if captured.ID != 7 || captured.Slug != "hello" {
		t.Errorf("got %+v", captured)
	}
}

func TestBindPath_InvalidReturns400(t *testing.T) {
	type Params struct {
		ID int `path:"id"`
	}

	r := chi.NewRouter()
	var bindErr error
	GET(r, "/items/{id}", func(w http.ResponseWriter, req *http.Request) error {
		var p Params
		bindErr = BindPath(req, &p)
		return NoContent(w)
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/items/abc", nil))

	he := asHTTPError(t, bindErr)
	if he.Status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", he.Status)
	}
}

func TestBindHeader(t *testing.T) {
	type Params struct {
		Token   string `header:"Authorization"`
		Version int    `header:"X-Api-Version"`
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer abc")
	req.Header.Set("X-Api-Version", "2")

	var p Params
	if err := BindHeader(req, &p); err != nil {
		t.Fatalf("BindHeader: %v", err)
	}
	if p.Token != "Bearer abc" || p.Version != 2 {
		t.Errorf("got %+v", p)
	}
}

func TestBindHeader_MissingRequired422(t *testing.T) {
	type Params struct {
		Token string `header:"Authorization" validate:"required"`
	}
	req := httptest.NewRequest("GET", "/", nil)
	var p Params
	he := asHTTPError(t, BindHeader(req, &p))
	if he.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", he.Status)
	}
}

func TestBindJSON_Valid(t *testing.T) {
	type Body struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Jo","email":"jo@x.com"}`))
	var b Body
	if err := BindJSON(req, &b); err != nil {
		t.Fatalf("BindJSON: %v", err)
	}
	if b.Name != "Jo" || b.Email != "jo@x.com" {
		t.Errorf("got %+v", b)
	}
}

func TestBindJSON_InvalidSyntax400(t *testing.T) {
	type Body struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":`))
	var b Body
	he := asHTTPError(t, BindJSON(req, &b))
	if he.Status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", he.Status)
	}
}

func TestBindJSON_ValidationFailure422(t *testing.T) {
	type Body struct {
		Email string `json:"email" validate:"required,email"`
	}
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"email":"not-an-email"}`))
	var b Body
	he := asHTTPError(t, BindJSON(req, &b))
	if he.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", he.Status)
	}
}

func TestBindJSON_NilBodyStillValidates(t *testing.T) {
	type Body struct {
		Name string `json:"name" validate:"required"`
	}
	req := httptest.NewRequest("POST", "/", nil)
	var b Body

	he := asHTTPError(t, BindJSON(req, &b))
	if he.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", he.Status)
	}
}

func TestBindJSON_EmptyBodyStillValidates(t *testing.T) {
	type Body struct {
		Name string `json:"name" validate:"required"`
	}
	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	var b Body
	he := asHTTPError(t, BindJSON(req, &b))
	if he.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", he.Status)
	}
}

func TestBindJSON_BodyTooLarge413(t *testing.T) {
	t.Cleanup(func() { SetMaxBodyBytes(4 << 20) })
	SetMaxBodyBytes(10)

	type Body struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"way too long to fit"}`))
	var b Body
	he := asHTTPError(t, BindJSON(req, &b))
	if he.Status != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", he.Status)
	}
}

func TestBindJSON_ExactLimitAccepted(t *testing.T) {
	t.Cleanup(func() { SetMaxBodyBytes(4 << 20) })

	body := `{"n":1}`
	SetMaxBodyBytes(int64(len(body)))

	type Body struct {
		N int `json:"n"`
	}
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	var b Body
	if err := BindJSON(req, &b); err != nil {
		t.Fatalf("body at exact limit should be accepted, got %v", err)
	}
	if b.N != 1 {
		t.Errorf("N = %d, want 1", b.N)
	}
}

func TestBind_CombinesPathQueryHeader(t *testing.T) {
	type Params struct {
		ID    int    `path:"id"`
		Q     string `query:"q"`
		Trace string `header:"X-Trace"`
	}

	r := chi.NewRouter()
	var captured Params
	var bindErr error
	GET(r, "/x/{id}", func(w http.ResponseWriter, req *http.Request) error {
		bindErr = Bind(req, &captured)
		return NoContent(w)
	})

	req := httptest.NewRequest("GET", "/x/9?q=search", nil)
	req.Header.Set("X-Trace", "t-123")
	r.ServeHTTP(httptest.NewRecorder(), req)

	if bindErr != nil {
		t.Fatalf("Bind: %v", bindErr)
	}
	if captured.ID != 9 || captured.Q != "search" || captured.Trace != "t-123" {
		t.Errorf("got %+v", captured)
	}
}

func TestDecode_NonPointerIsNoop(t *testing.T) {
	type Params struct {
		Name string `query:"name"`
	}

	req := httptest.NewRequest("GET", "/?name=x", nil)
	if err := BindQuery(req, Params{}); err != nil {
		t.Fatalf("expected nil error for non-pointer dst, got %v", err)
	}
}

func TestDecode_UnsupportedTagsAndUnexported(t *testing.T) {
	type Params struct {
		Skip     string `query:"-"`
		NoTag    string
		exported string `query:"secret"`
		Name     string `query:"name"`
	}
	req := httptest.NewRequest("GET", "/?name=ok&secret=nope", nil)
	var p Params
	if err := BindQuery(req, &p); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}
	if p.Name != "ok" {
		t.Errorf("Name = %q, want ok", p.Name)
	}
	if p.Skip != "" || p.NoTag != "" || p.exported != "" {
		t.Errorf("fields without a usable tag were populated: %+v", p)
	}
}

func TestSetScalar_PointerFields(t *testing.T) {
	type Params struct {
		Age *int `query:"age"`
	}
	req := httptest.NewRequest("GET", "/?age=5", nil)
	var p Params
	if err := BindQuery(req, &p); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}
	if p.Age == nil || *p.Age != 5 {
		t.Errorf("Age = %v, want pointer to 5", p.Age)
	}
}

func TestSetScalar_UintAndFloatVariants(t *testing.T) {
	type Params struct {
		U8  uint8   `query:"u8"`
		I64 int64   `query:"i64"`
		F32 float32 `query:"f32"`
	}
	req := httptest.NewRequest("GET", "/?u8=200&i64=-9000000000&f32=1.5", nil)
	var p Params
	if err := BindQuery(req, &p); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}
	if p.U8 != 200 || p.I64 != -9000000000 || p.F32 != 1.5 {
		t.Errorf("got %+v", p)
	}
}
