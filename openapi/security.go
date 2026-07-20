package openapi

// SecurityScheme represents an OpenAPI security scheme.
// Use the constructor functions (BearerAuth, BasicAuth, etc.) to create one.
type SecurityScheme struct {
	Type             string      `json:"type"`
	Description      string      `json:"description,omitempty"`
	Scheme           string      `json:"scheme,omitempty"`
	BearerFormat     string      `json:"bearerFormat,omitempty"`
	In               string      `json:"in,omitempty"`
	Name             string      `json:"name,omitempty"`
	Flows            *OAuthFlows `json:"flows,omitempty"`
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty"`
}

// APIKeyLocation specifies where an API key is sent (header, query, or cookie).
type APIKeyLocation string

const (
	InHeader APIKeyLocation = "header"
	InQuery  APIKeyLocation = "query"
	InCookie APIKeyLocation = "cookie"
)

// OAuthFlows holds the four OAuth 2.0 flow types.
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// OAuthFlow represents a single OAuth 2.0 flow.
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

// SecurityRequirement maps scheme names to required scopes.
// Use Require or RequireScopes to build one.
type SecurityRequirement map[string][]string

// BearerAuth creates an HTTP Bearer token security scheme.
// Optionally specify the bearer format (e.g. "JWT").
func BearerAuth(bearerFormat ...string) SecurityScheme {
	s := SecurityScheme{Type: "http", Scheme: "bearer"}
	if len(bearerFormat) > 0 && bearerFormat[0] != "" {
		s.BearerFormat = bearerFormat[0]
	}
	return s
}

// BasicAuth creates an HTTP Basic authentication security scheme.
func BasicAuth() SecurityScheme {
	return SecurityScheme{Type: "http", Scheme: "basic"}
}

// APIKeyAuth creates an API key security scheme.
//
//	name := openapi.APIKeyAuth("X-API-Key", openapi.InHeader)
func APIKeyAuth(name string, in APIKeyLocation) SecurityScheme {
	return SecurityScheme{Type: "apiKey", Name: name, In: string(in)}
}

// OAuth2 creates an OAuth 2.0 security scheme with the given flows.
func OAuth2(flows OAuthFlows) SecurityScheme {
	return SecurityScheme{Type: "oauth2", Flows: &flows}
}

// OpenIDConnect creates an OpenID Connect security scheme.
func OpenIDConnect(discoveryURL string) SecurityScheme {
	return SecurityScheme{Type: "openIdConnect", OpenIDConnectURL: discoveryURL}
}

// Require creates a security requirement for the given scheme names (no scopes).
func Require(schemes ...string) SecurityRequirement {
	req := make(SecurityRequirement, len(schemes))
	for _, s := range schemes {
		req[s] = []string{}
	}
	return req
}

// RequireScopes creates a security requirement with specific OAuth scopes.
func RequireScopes(scheme string, scopes ...string) SecurityRequirement {
	return SecurityRequirement{scheme: scopes}
}
