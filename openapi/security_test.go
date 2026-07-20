package openapi

import (
	"reflect"
	"testing"
)

func TestBearerAuth(t *testing.T) {
	s := BearerAuth()
	if s.Type != "http" || s.Scheme != "bearer" {
		t.Errorf("got %+v", s)
	}
	if s.BearerFormat != "" {
		t.Errorf("BearerFormat = %q, want empty", s.BearerFormat)
	}

	withFmt := BearerAuth("JWT")
	if withFmt.BearerFormat != "JWT" {
		t.Errorf("BearerFormat = %q, want JWT", withFmt.BearerFormat)
	}

	empty := BearerAuth("")
	if empty.BearerFormat != "" {
		t.Errorf("BearerFormat = %q, want empty for empty arg", empty.BearerFormat)
	}
}

func TestBasicAuth(t *testing.T) {
	s := BasicAuth()
	if s.Type != "http" || s.Scheme != "basic" {
		t.Errorf("got %+v", s)
	}
}

func TestAPIKeyAuth(t *testing.T) {
	s := APIKeyAuth("X-API-Key", InHeader)
	if s.Type != "apiKey" || s.Name != "X-API-Key" || s.In != "header" {
		t.Errorf("got %+v", s)
	}
	if APIKeyAuth("k", InQuery).In != "query" {
		t.Error("InQuery should map to 'query'")
	}
	if APIKeyAuth("k", InCookie).In != "cookie" {
		t.Error("InCookie should map to 'cookie'")
	}
}

func TestOAuth2(t *testing.T) {
	flows := OAuthFlows{
		AuthorizationCode: &OAuthFlow{
			AuthorizationURL: "https://auth",
			TokenURL:         "https://token",
			Scopes:           map[string]string{"read": "Read access"},
		},
	}
	s := OAuth2(flows)
	if s.Type != "oauth2" {
		t.Errorf("Type = %q, want oauth2", s.Type)
	}
	if s.Flows == nil || s.Flows.AuthorizationCode == nil {
		t.Fatal("flows not carried")
	}
	if s.Flows.AuthorizationCode.TokenURL != "https://token" {
		t.Errorf("TokenURL = %q", s.Flows.AuthorizationCode.TokenURL)
	}
}

func TestOpenIDConnect(t *testing.T) {
	s := OpenIDConnect("https://issuer/.well-known/openid-configuration")
	if s.Type != "openIdConnect" {
		t.Errorf("Type = %q, want openIdConnect", s.Type)
	}
	if s.OpenIDConnectURL != "https://issuer/.well-known/openid-configuration" {
		t.Errorf("OpenIDConnectURL = %q", s.OpenIDConnectURL)
	}
}

func TestRequire(t *testing.T) {
	req := Require("a", "b")
	if len(req) != 2 {
		t.Fatalf("len = %d, want 2", len(req))
	}
	if req["a"] == nil || len(req["a"]) != 0 {
		t.Errorf("req[a] = %v, want empty non-nil slice", req["a"])
	}
}

func TestRequireScopes(t *testing.T) {
	req := RequireScopes("oauth", "read", "write")
	want := []string{"read", "write"}
	if !reflect.DeepEqual(req["oauth"], want) {
		t.Errorf("scopes = %v, want %v", req["oauth"], want)
	}
}
