package openapi

type document struct {
	OpenAPI    string                `json:"openapi"`
	Info       info                  `json:"info"`
	Servers    []Server              `json:"servers,omitempty"`
	Security   []SecurityRequirement `json:"security,omitempty"`
	Paths      map[string]pathItem   `json:"paths,omitempty"`
	Components *components           `json:"components,omitempty"`
}

type info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Server represents an OpenAPI server object.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type pathItem map[string]*Operation

// Operation represents an OpenAPI path operation (GET, POST, etc.).
type Operation struct {
	OperationID string                `json:"operationId,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   Responses             `json:"responses"`
	Security    []SecurityRequirement `json:"security,omitempty"`
}

// Parameter represents an OpenAPI parameter (path, query, or header).
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // "path", "query", "header"
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Deprecated  bool    `json:"deprecated,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
	Example     any     `json:"example,omitempty"`
}

// RequestBody represents an OpenAPI request body.
type RequestBody struct {
	Required    bool                 `json:"required,omitempty"`
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
}

// MediaType describes the schema and example for a request/response body.
type MediaType struct {
	Schema  *Schema `json:"schema,omitempty"`
	Example any     `json:"example,omitempty"`
}

// Responses maps status codes to response definitions.
type Responses map[string]*Response

// Response represents an OpenAPI response.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type components struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

// Schema represents an OpenAPI schema object.
// Generated automatically from Go types via the Spec builder.
type Schema struct {
	Ref string `json:"$ref,omitempty"`

	Type        any    `json:"type,omitempty"`   // string or []string (OpenAPI 3.1 nullable)
	Format      string `json:"format,omitempty"` // "int32","int64","float","double","email","uuid","uri","date-time"
	Description string `json:"description,omitempty"`
	Example     any    `json:"example,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`

	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	AdditionalProperties any                `json:"additionalProperties,omitempty"`

	Items    *Schema `json:"items,omitempty"`
	MinItems *int    `json:"minItems,omitempty"`
	MaxItems *int    `json:"maxItems,omitempty"`

	MinLength *int   `json:"minLength,omitempty"`
	MaxLength *int   `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	Enum      []any  `json:"enum,omitempty"`

	Minimum          *float64 `json:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`
	MultipleOf       *float64 `json:"multipleOf,omitempty"`
}
