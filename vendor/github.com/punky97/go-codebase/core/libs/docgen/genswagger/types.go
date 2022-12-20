package genswagger

import (
	"reflect"
)

type SwaggerServerInfo struct {
	ServiceName string
	Version     string
	Name        string
	BasePath    string
	Description string
	RoutesInfo  []SwaggerRouteInfo
}

type SwaggerRouteInfo struct {
	RequestParams  reflect.Type
	RequestBody    reflect.Type
	ResponseBody   reflect.Type
	Pattern        string
	Method         string
	Name           string
	Description    string
	ResponseDes    string
	RequestDes     string
	Tags           []string
	Deprecated     bool
	Summary        string
	AuthInfo       AuthInfo
	HandlerName    string
	HandlerPkgName string
}

type AuthInfo struct {
	Enable         bool
	TokenType      string
	RequiredFields []string
	RestrictScopes []string
}

// http://swagger.io/specification/#infoObject
type SwaggerInfoObject struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	TermsOfService string `json:"termsOfService,omitempty"`
	Version        string `json:"version"`

	Contact *SwaggerContactObject `json:"contact,omitempty"`
	License *SwaggerLicenseObject `json:"license,omitempty"`
}

// http://swagger.io/specification/#contactObject
type SwaggerContactObject struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// http://swagger.io/specification/#licenseObject
type SwaggerLicenseObject struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// http://swagger.io/specification/#externalDocumentationObject
type SwaggerExternalDocumentationObject struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

type Swagger struct {
	SwaggerObject
}

// http://swagger.io/specification/#swaggerObject
type SwaggerObject struct {
	Swagger             string                              `json:"swagger"`
	Info                SwaggerInfoObject                   `json:"info"`
	Host                string                              `json:"host,omitempty"`
	BasePath            string                              `json:"basePath,omitempty"`
	Paths               SwaggerPathsObject                  `json:"paths"`
	Schemes             []string                            `json:"schemes"`
	Consumes            []string                            `json:"consumes"`
	Produces            []string                            `json:"produces"`
	Definitions         Definitions                         `json:"definitions"`
	StreamDefinitions   Definitions                         `json:"x-stream-definitions,omitempty"`
	SecurityDefinitions SwaggerSecurityDefinitionsObject    `json:"securityDefinitions,omitempty"`
	Security            []SwaggerSecurityRequirementObject  `json:"security,omitempty"`
	ExternalDocs        *SwaggerExternalDocumentationObject `json:"externalDocs,omitempty"`
	Tags                []SwaggerTag                        `json:"tags,omitempty"`
}

// http://swagger.io/specification/#securityDefinitionsObject
type SwaggerSecurityDefinitionsObject map[string]*SwaggerSecuritySchemeObject

// http://swagger.io/specification/#securitySchemeObject
type SwaggerSecuritySchemeObject struct {
	Type             string              `json:"type"`
	Description      string              `json:"description,omitempty"`
	Name             string              `json:"name,omitempty"`
	In               string              `json:"in,omitempty"`
	Flow             string              `json:"flow,omitempty"`
	AuthorizationURL string              `json:"authorizationUrl,omitempty"`
	TokenURL         string              `json:"tokenUrl,omitempty"`
	Scopes           swaggerScopesObject `json:"scopes,omitempty"`
}

// http://swagger.io/specification/#scopesObject
type swaggerScopesObject map[string]string

// http://swagger.io/specification/#securityRequirementObject
type SwaggerSecurityRequirementObject map[string][]string

// http://swagger.io/specification/#pathsObject
type SwaggerPathsObject map[string]SwaggerPathItemObject

// http://swagger.io/specification/#pathItemObject
type SwaggerPathItemObject struct {
	Get    *SwaggerOperationObject `json:"get,omitempty"`
	Delete *SwaggerOperationObject `json:"delete,omitempty"`
	Post   *SwaggerOperationObject `json:"post,omitempty"`
	Put    *SwaggerOperationObject `json:"put,omitempty"`
	Patch  *SwaggerOperationObject `json:"patch,omitempty"`
}

// http://swagger.io/specification/#operationObject
type SwaggerOperationObject struct {
	Summary     string                  `json:"summary,omitempty"`
	Description string                  `json:"description,omitempty"`
	Responses   SwaggerResponsesObject  `json:"responses"`
	Parameters  SwaggerParametersObject `json:"parameters,omitempty"`
	Tags        []string                `json:"tags,omitempty"`
	Deprecated  bool                    `json:"deprecated,omitempty"`
	ID          string                  `json:"operationId,omitempty"`
	Consumes    []string                `json:"consumes,omitempty"`
	Produces    []string                `json:"produces,omitempty"`

	Security     []SwaggerSecurityRequirementObject  `json:"security,omitempty"`
	ExternalDocs *SwaggerExternalDocumentationObject `json:"externalDocs,omitempty"`
}

type SwaggerParametersObject []SwaggerParameterObject

// http://swagger.io/specification/#parameterObject
type SwaggerParameterObject struct {
	CommonValidations
	Name             string        `json:"name"`
	Description      string        `json:"description,omitempty"`
	In               string        `json:"in,omitempty"`
	Required         bool          `json:"required"`
	Type             string        `json:"type,omitempty"`
	Format           string        `json:"format,omitempty"`
	Items            *Type         `json:"items,omitempty"`
	Enum             []interface{} `json:"enum,omitempty"`
	CollectionFormat string        `json:"collectionFormat,omitempty"`
	Default          interface{}   `json:"default,omitempty"`
	MinItems         *int          `json:"minItems,omitempty"`
	Example          interface{}   `json:"example,omitempty"`

	// Or you can explicitly refer to another type. If this is defined all
	// other fields should be empty
	Schema *Type `json:"schema,omitempty"`
}

// CommonValidations describe common JSON-schema validations
type CommonValidations struct {
	Maximum          *float64      `json:"maximum,omitempty"`
	ExclusiveMaximum bool          `json:"exclusiveMaximum,omitempty"`
	Minimum          *float64      `json:"minimum,omitempty"`
	ExclusiveMinimum bool          `json:"exclusiveMinimum,omitempty"`
	MaxLength        *int64        `json:"maxLength,omitempty"`
	MinLength        *int64        `json:"minLength,omitempty"`
	Pattern          string        `json:"pattern,omitempty"`
	MaxItems         *int64        `json:"maxItems,omitempty"`
	MinItems         *int64        `json:"minItems,omitempty"`
	UniqueItems      bool          `json:"uniqueItems,omitempty"`
	MultipleOf       *float64      `json:"multipleOf,omitempty"`
	Enum             []interface{} `json:"enum,omitempty"`
}

type Header struct {
	CommonValidations
	Description string `json:"description,omitempty"`
	Type
}

// http://swagger.io/specification/#responsesObject
type SwaggerResponsesObject map[int]SwaggerResponseObject

// http://swagger.io/specification/#responseObject
type SwaggerResponseObject struct {
	Description string            `json:"description"`
	Schema      *Type             `json:"schema"`
	Headers     map[string]Header `json:"headers,omitempty"`
}

// TagProps describe a tag entry in the top level tags section of a swagger spec
type SwaggerTag struct {
	Description  string                 `json:"description,omitempty"`
	Name         string                 `json:"name,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

type ExternalDocumentation struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

const (
	basic       = "basic"
	apiKey      = "apiKey"
	oauth2      = "oauth2"
	implicit    = "implicit"
	password    = "password"
	application = "application"
	accessCode  = "accessCode"
)

// BasicAuth creates a basic auth security scheme
func BasicAuth() *SwaggerSecuritySchemeObject {
	return &SwaggerSecuritySchemeObject{Type: basic}
}

// APIKeyAuth creates an api key auth security scheme
func APIKeyAuth(fieldName, valueSource string) *SwaggerSecuritySchemeObject {
	return &SwaggerSecuritySchemeObject{Type: apiKey, Name: fieldName, In: valueSource}
}

// OAuth2Application creates an application flow oauth2 security scheme
func OAuth2Application(tokenURL string) *SwaggerSecuritySchemeObject {
	return &SwaggerSecuritySchemeObject{
		Type:     oauth2,
		Flow:     application,
		TokenURL: tokenURL,
	}
}

// AddScope adds a scope to this security scheme
func (s *SwaggerSecuritySchemeObject) AddScope(scope, description string) {
	if s.Scopes == nil {
		s.Scopes = make(map[string]string)
	}
	s.Scopes[scope] = description
}

// OAuth2Implicit creates an implicit flow oauth2 security scheme
func OAuth2Implicit(authorizationURL string) *SwaggerSecuritySchemeObject {
	return &SwaggerSecuritySchemeObject{
		Type:             oauth2,
		Flow:             implicit,
		AuthorizationURL: authorizationURL,
	}
}

// OAuth2Password creates a password flow oauth2 security scheme
func OAuth2Password(tokenURL string) *SwaggerSecuritySchemeObject {
	return &SwaggerSecuritySchemeObject{
		Type:     oauth2,
		Flow:     password,
		TokenURL: tokenURL,
	}
}

// OAuth2AccessToken creates an access token flow oauth2 security scheme
func OAuth2AccessToken(authorizationURL, tokenURL string) *SwaggerSecuritySchemeObject {
	return &SwaggerSecuritySchemeObject{
		Type:             oauth2,
		Flow:             accessCode,
		AuthorizationURL: authorizationURL,
		TokenURL:         tokenURL,
	}
}

// StringOrArray represents a value that can either be a string
// or an array of strings. Mainly here for serialization purposes
type StringOrArray []string
