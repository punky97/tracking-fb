package registry

// Service --
type Service struct {
	Name      string            `json:"name,omitempty"`
	IDName    string            `json:",omitempty"`
	Version   string            `json:"version,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Endpoints []*Endpoint       `json:"endpoints,omitempty"`
	Nodes     []*Node           `json:"nodes,omitempty"`
}

// Node --
type Node struct {
	Id       string            `json:"id,omitempty"`
	Address  string            `json:"address,omitempty"`
	Port     int               `json:"port,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Endpoint --
type Endpoint struct {
	Name     string            `json:"name,omitempty"`
	Request  *Value            `json:"request,omitempty"`
	Response *Value            `json:"response,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	AuthInfo *AuthInfo         `json:"authinfo,omitempty"`
	Timeout  int64             `json:"timeout,omitempty"`
}

// AuthInfo --
type AuthInfo struct {
	Enable         bool     `json:"enable,omitempty"`
	TokenType      string   `json:"token_type,omitempty"`
	RequiredFields []string `json:"required_fields,omitempty"`
	RestrictScopes []string `json:"restrict_scopes,omitempty"`
}

// Value --
type Value struct {
	Name   string   `json:"name,omitempty"`
	Type   string   `json:"type,omitempty"`
	Values []*Value `json:"values,omitempty"`
}
