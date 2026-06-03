package openapi

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

type APIKeyLocation string

const (
	InHeader APIKeyLocation = "header"
	InQuery  APIKeyLocation = "query"
	InCookie APIKeyLocation = "cookie"
)

type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

type SecurityRequirement map[string][]string

func BearerAuth(bearerFormat ...string) SecurityScheme {
	s := SecurityScheme{Type: "http", Scheme: "bearer"}
	if len(bearerFormat) > 0 && bearerFormat[0] != "" {
		s.BearerFormat = bearerFormat[0]
	}
	return s
}

func BasicAuth() SecurityScheme {
	return SecurityScheme{Type: "http", Scheme: "basic"}
}

func APIKeyAuth(name string, in APIKeyLocation) SecurityScheme {
	return SecurityScheme{Type: "apiKey", Name: name, In: string(in)}
}

func OAuth2(flows OAuthFlows) SecurityScheme {
	return SecurityScheme{Type: "oauth2", Flows: &flows}
}

func OpenIDConnect(discoveryURL string) SecurityScheme {
	return SecurityScheme{Type: "openIdConnect", OpenIDConnectURL: discoveryURL}
}

func Require(schemes ...string) SecurityRequirement {
	req := make(SecurityRequirement, len(schemes))
	for _, s := range schemes {
		req[s] = []string{}
	}
	return req
}

func RequireScopes(scheme string, scopes ...string) SecurityRequirement {
	return SecurityRequirement{scheme: scopes}
}
